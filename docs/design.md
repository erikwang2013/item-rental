# 项目设计 — 多端租赁平台 item-rental

> 本地设计文档(仓库内长期保留)。规划见 [plan.md](plan.md),任务见 [task.md](task.md)。

## 1. 技术栈

| 组件 | 选型 | 用途 |
| --- | --- | --- |
| Web 框架 | Beego v2.3.10(MVC) | 路由/控制器/ORM |
| 语言 | Go 1.24 | — |
| 主库 | MySQL 8 | users/categories/items/orders/payments |
| 缓存 | Redis(可选) | refresh 会话轮换 |
| 支付 | 微信支付 V3(JSAPI, M3) | 预付、回调、退款 |
| 隐私 | 手机号 HMAC-SHA256 + 实名 AES-GCM | PII 存储保护 |

## 2. 架构分层

```
客户端(Flutter / 微信小程序)
  │  HTTPS /api/v1
接入层  ─ Beego Router(方法感知 JWT 过滤器)· CORS
业务层  ─ Controllers(统一 OK/Fail 信封)
         ─ Services(纯业务逻辑,免 ORM 可单测)
         ─ 集成网关 payments/ · models/ · middleware/
数据层  ─ MySQL / Redis(可选) / static/uploads / 微信支付
```

详见 [docs/architecture.svg](architecture.svg)。

## 3. 核心设计决策

### 3.1 订单 7 态状态机

```
0 待支付 → 1 待取 → 2 租赁中 → 3 待归还 → 4 已归还(终态)
  │                       │
  └→ 5 已取消(0→5,退款)   └→ 6 违约(3→6)
```

- **迁移约束**:每次迁移 = 条件更新(`Filter(status=期望前置状态).Update`),`n==0` 视为已被并行请求处理,幂等返回。
- **资金动作**:0→1 收租金;0→5 退款;其余迁移仅做状态流转。
- 代码映射:`server/models/order.go`(OrderStatusPending=0…Breach=6)、`server/services/orderflow.go`。

### 3.2 资金路径(必守)

- 全部条件更新幂等;notify 回调重复投递安全。
- 微信退款 real 模式需商户证书(双证书),CI 一律 mock 短路。
- 退款守卫:仅 status=1(待取)可退,租赁中/已归还/违约一律 409。
- 拒绝状态**:不下单给下架/存量不足/renter==owner 的物品。

### 3.3 认证

- 手机号直接登录(mock 入口,新用户自动注册),无短信验证码链路。
- JWT:Access(2h)+ Refresh(7d);refresh 轮换支持多端并存、单端登出。
- refresh 会话默认不落 Redis(`redis_enabled=false`),多实例部署置 `true` 落到 `auth:refresh:{uid}` hash。
- PII:phone 存 HMAC-SHA256 hex;real_name AES-GCM 加密存储,密钥 `ITEM_RENTAL_PII_KEY` 缺失 fail-fast。

### 3.4 安全与边界

- JWT 方法感知过滤器:公开 GET 与需登录写操作分流;owner/参与者权限在 controller 校验。
- 边界校验:金额/长度/图片/手机号/日期,一律在边界收口。
- snowflake 主键 > 2^53,JSON 一律字符串输出(JS 安全)。

### 3.5 上传

- 头像:`POST /user/avatar`,单文件 ≤4MB,落 `static/uploads/avatars/`。
- 物品图:`POST /items/upload`,多文件 ≤9×4MB,落 `static/uploads/items/`。
- 文件名由服务端生成(不信任客户端文件名),扩展名白名单 jpg/jpeg/png/webp。

## 4. 数据模型

- **users**: Id | Phone(唯一,HMAC-SHA256) | Nickname | Avatar | RealName(AES-GCM) | Status
- **categories**: Id | Name | ParentId | Icon | Sort | Status
- **items**: Id | OwnerId | CategoryId | Title | Desc | Images | DailyPrice | Stock | Status | City
- **orders**: Id | OrderNo(唯一) | ItemId | RenterId | OwnerId | StartDate | EndDate | Days | RentAmount | Status | PayTradeNo | CancelReason
- **payments**: 支付台账(预付/回调/退款记录)

模型↔Schema 漂移与配置覆盖见 [docs/config-runbook.md](config-runbook.md)。

## 5. API 契约(主线)

| 方法 | 路径 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| POST | /auth/login | 公开 | 手机号登录(自动注册) |
| POST | /auth/refresh | 公开 | 双 token 轮换 |
| POST | /auth/logout | JWT | 登出/撤销会话 |
| GET/PUT | /user/profile | JWT | 用户资料 |
| POST | /user/avatar | JWT | 头像上传 |
| GET | /categories | 公开 | 类目列表 |
| GET | /items、/items/:id | 公开 | 分页列表/详情 |
| POST | /items、/items/upload | JWT | 发布/图片上传 |
| PUT | /items/:id | JWT | 修改(owner) |
| POST | /items/:id/offshelf | JWT | 下架(owner) |
| POST | /orders | JWT | 创建订单(status=0) |
| GET | /orders、/orders/:id | JWT | 我的订单(renter/owner) |
| POST | /orders/:id/pickup \| return_request \| return_confirm \| breach \| cancel | JWT | 状态迁移 |
| POST | /pay/unifiedorder、/pay/refund | JWT | 预付/退款 |
| POST | /pay/notify | 签名回调 | 支付/退款结果回调 |

## 6. 排查要点(Tech Notes)

- **3306 占用**:dev 用 `MYSQL_HOST_PORT=3307` 覆盖;宿主机 3307 不可用时同理上调。
- **资金断言**:任何「应该只有一次」的写入,用 `Filter(status=…)` 而非先查后写。
- **snowflake id**:客户端必须按字符串处理;服务端请求体 item_id/order_id 收字符串。
- **PII 启动失败**:检查 `ITEM_RENTAL_PII_KEY` 是否为 64 位 hex。

## 7. 相关文档

- [plan.md](plan.md) — 规划与路线图
- [task.md](task.md) — 任务清单
- [config-runbook.md](config-runbook.md) — 配置与部署
- [frontend-plan.md](frontend-plan.md) — 前端规划(仅规划)
