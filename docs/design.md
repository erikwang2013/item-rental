# 项目设计 — 多端租赁平台 item-rental

> 本地设计文档(仓库内长期保留)。规划见 [plan.md](plan.md),任务见 [task.md](task.md)。

## 1. 技术栈

| 组件 | 选型 | 用途 |
| --- | --- | --- |
| Web 框架 | Beego v2.3.10(MVC) | 路由/控制器/ORM |
| 语言 | Go 1.24 | — |
| 主库 | MySQL 8 | users/items/orders/deposits/payments |
| 缓存 | Redis | 验证码限频、refresh 轮换 |
| 搜索引擎 | OpenSearch 2.14(security 启用) | 物品全文检索、坐标索引 |
| 检索驱动 | go-scout(Searchable 抽象) | ES/OpenSearch 客户端 |
| 支付 | 微信支付 V3(JSAPI, M3) | 预付、回调、退款 |
| 安全 | security-go 中间件 | 攻击检测、IP 封禁 |

## 2. 架构分层

```
客户端(小程序 / H5 / 管理后台)
  │  HTTPS /api/v1
接入层  ─ Beego Router(方法感知过滤器)
         ─ security-go(GET /items/:id 公开放行)
         ─ JWT 双 Token(Access 2h + Refresh 7d 轮换)
业务层  ─ Controllers(统一 OK/Fail 信封)
         ─ Services(纯业务逻辑,免 ORM 可单测)
         ─ 集成网关 payments/· search/· middleware/·
数据层  ─ MySQL / Redis / OpenSearch / 微信支付
```

详见 [docs/architecture.svg](architecture.svg)。

## 3. 核心设计决策

### 3.1 订单 7 态状态机

```
0 待支付 → 1 待取 → 2 租赁中 → 3 待归还 → 4 已归还(终态)
  │                      │
  └→ 5 已取消(0→5,退款) └→ 6 违约(3→6,扣押金)
```

- **迁移约束**:每次迁移 = 条件更新(`Filter(status=期望前置状态).Update`),`n==0` 视为已被并行请求处理,幂等返回。
- **资金动作绑定**:0→1 收租金;1→2 冻结押金(type 1);3→4 解冻押金(type 2)并结算;3→6 扣款(type 3);0→5 退款。
- 代码映射:`server/models/order.go`(OrderStatusPending=0…Breach=6)、`server/services/orderflow.go`。

### 3.2 押金台账 deposits

| 字段 | 说明 |
| --- | --- |
| OrderId | 关联订单 |
| Type | 1 冻结 / 2 解冻 / 3 扣款 |
| Amount | decimal(12,2) |
| BalAfter | 变动后余额快照(审计) |
| CreatedAt | 时间戳 |

与 `users.deposit_bal` 同步增减,台账为事实源。

### 3.3 资金路径(必守)

- 全部条件更新幂等;notify 回调重复投递安全。
- 微信退款 real 模式需商户证书(双证书),CI 一律 mock 短路。
- 拒绝状态**:不下单给下架/存量不足/renter==owner 的物品。

### 3.4 认证

- 短信:mock(不入 Redis)/ real(Redis `sms:limit:{phone}` INCR+TTL 60s 限频);响应永不返回明文验证码。
- JWT:Access(2h)+ Refresh(7d);refresh 轮换 = Redis `auth:refresh:{uid}` 单活跃 hash,新发双 token 覆写旧 hash。

### 3.5 检索

- 索引文档 = 上架物品;下架走 UnsearchableSync(真删除)。
- 启动 `ReindexAll` log-&-continue,OpenSearch 不可用时(dev)不 panic。
- 地理:**边界盒(WhereRange)+ Go Haversine 精滤**。
  `ponytail:` 大半径/距离排序需 OpenSearch geo_distance 时再升级。

### 3.6 安全

- security-go:`POST /user/profile` 等曾触发 403「恶意请求已拦截」= app 自身中间件(IP 封禁/请求特征),非基础设施故障(待联调时定位根因)。
- 边界校验:金额/坐标/长度/类别存在性,一律在边界收口。

## 4. 数据模型

- **users**: Id | Phone(唯一) | Nickname | Avatar | RealName | CreditScore | DepositBal(decimal(12,2)) | Status
- **items**: Id | Title | CategoryId | OwnerId | DailyPrice | Deposit | Stock | City | Lat | Lng | Images | Status(上下架)
- **orders**: Id | OrderNo(唯一) | ItemId | RenterId | OwnerId | StartDate | EndDate | Days | RentAmount | Deposit | Status | PayTradeNo | CancelReason | CreatedAt | UpdatedAt
- **deposits**: 见 3.2
- **payments**: 台账(预付/回调/退款记录)

模型↔Schema 漂移与配置覆盖见 [docs/config-runbook.md](config-runbook.md)。

## 5. API 契约(主线)

| 方法 | 路径 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| POST | /auth/sms | 公开 | 发验证码 |
| POST | /auth/login | 公开 | 验证码登录(自动注册) |
| POST | /auth/refresh | 公开 | 双 token 轮换 |
| GET/PUT | /user/profile | JWT | 用户资料 |
| GET | /categories | 公开 | 类目列表 |
| GET | /items、/items/search | 公开 | 分页/关键词/半径 |
| GET | /items/:id | 公开 | 物品详情 |
| POST | /items | JWT | 发布 |
| PUT | /items/:id | JWT | 修改(owner) |
| POST | /items/:id/offshelf | JWT | 下架(owner) |
| POST | /orders | JWT | 创建订单(status=0) |
| GET | /orders、/orders/:id | JWT | 我的订单(renter/owner) |
| POST | /orders/:id/pickup \| return_request \| return_confirm \| breach \| cancel | JWT | 状态迁移 |
| POST | /pay/unifiedorder、/pay/refund | JWT | 预付/退款 |
| POST | /pay/notify | 签名回调 | 支付/退款结果回调 |

## 6. 排查要点(Tech Notes)

- **403 恶意请求**:先查 `server/security/` 中间件触发条件与 IP 封禁窗口,勿误判基建。
- **OpenSearch 起不来**:2.14 镜像 security 已启用,compose **不得**再设 `plugins.security.disabled`;需 `${OPENSEARCH_ADMIN_PASSWORD:?}`。
- **3306 占用**:dev 用 `MYSQL_HOST_PORT=3307` 覆盖;宿主机 3307 不可用时同理上调。
- **资金断言**:任何「应该只有一次」的写入,想 `Filter(status=…)` 而非先查后写。

## 7. 相关文档

- [plan.md](plan.md) — 规划与路线图
- [task.md](task.md) — 任务清单
- [config-runbook.md](config-runbook.md) — 配置与部署
- [frontend-plan.md](frontend-plan.md) — 前端规划(仅规划)