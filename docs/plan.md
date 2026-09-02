# 项目规划 — 多端租赁平台 item-rental

> 本地规划文档(仓库内长期保留)。执行状态与进展见 [task.md](task.md),技术决策见 [design.md](design.md)。

## 1. 项目定位

物品租赁平台**后端**服务(Beego v2 + MySQL + Redis + OpenSearch),面向小程序 / H5 / 管理后台多端。业务形态:

- 房东(物品 Owner)发布闲置物品,按天计价 + 押金;
- 租客(Renter)搜索、下单、支付租金、取件使用、按约归还;
- 平台管理订单 7 态状态机、押金台账、退款与违约处置。

**已确认执行决定(2026-09-01)**:P0 + P1 一并推进,P2 与前端规划另单独确认;前端**仅规划不实现**。

## 2. 现状基线(规划起点)

### 端点

`/health`;auth/sms、login、refresh;user/profile GET+PUT;categories GET;items GET(分页)/search/:id/POST/PUT/:id/offshelf;pay/unifiedorder、notify。规划时**无任何 orders 路由**(现已补齐)。

### 已证实的缺陷(规划时发现,现已修复)

- `POST /api/v1/items` Create 无 JWTAuth 但调用 GetUserID → 发布物品永远 401。
- `GET /api/v1/items/:id` 被 JWTAuth 过滤器误档 → 公开读设计要求被破坏。
- search RemoveItem 实为 upsert 而非删除 → 下架文档残留;`ReindexAll` 无启动重建调用。
- SendSms 硬编码返回明文验证码、无频率限制。
- Item.Create 校验缺失(Deposit/Stock/Lat/Lng/Images/长度);Update 无条件覆写 lat/lng。
- Refresh 不轮换 refresh token、无撤销。
- 退款彻底缺失(status 3 仅常量);deposits 台账零逻辑;lat/lng 死数据。

### 可复用模式(规划确认)

BaseController OK/Fail 统一信封;middleware.GetUserID;owner 归属检查(403);条件更新幂等(`Filter(status=期望值).Update`,`n==0`=已处理);payments/sign.go 完整签名器;security_middleware trusted-proxy IP 提取。

## 3. 产品终态

Schema 指向**双边 P2P 租赁市场**:用户带信用分 + 押金账户(deposit_bal);物品带坐标(半径检索);订单 7 态机(0待支付→1待取→2租赁中→3待归还→4已归还→5已取消→6违约);deposits 押金台账(1冻/2解冻/3扣款);payments 台账含退款。

**当前实现切片**:SMS-mock 认证、类目读、物品 CRUD + 关键词搜索、微信 M3 预付 + 回调置已付、订单全流程、押金、退款、地理检索、refresh 轮换。**未实现**:真实短信、前端(仅规划)、信用分逻辑、PII 加密。

## 4. 阶段路线图

| 阶段 | 内容 | 状态 |
| --- | --- | --- |
| **P0** 正确性门 | 修断点(公开读/Create/下架删除/短信泄码/校验空档)+ 订单创建主干 | ✅ 完成 |
| **P1** 领域完备 | 订单生命周期 + 押金台账、退款流、地理半径检索、测试缝、refresh 轮换 | ✅ 完成 |
| **P2** 加固 | 配置/密钥 bootstrap、schema 漂移、CI 门、前端规划书 | ✅ 完成 |
| **P3** 收尾 | README 与本地文档、推送/版本发布规则 | 🟡 本轮 |

**lazy 裁剪标记(已决策)**:

- 独立退款-notify 处理器 → 并入现有 notify 分支;
- OpenSearch 精确 geo_distance → 边界盒 + Go Haversine 精滤(lazy 近似,`ponytail:` 注明升级路径);
- refresh 撤销清单 → 只做单活跃轮换;
- golangci-lint 调参 → 用默认配置;
- 信用分与 PII 加密实现 → OUT-OF-SCOPE(仅 schema 保留字段)。

## 5. 定稿任务分解

| 编号 | 任务 | 层级 | 状态 |
| --- | --- | --- | --- |
| P0-1 | 路由鉴权纠正(router.go 方法感知过滤器) | cod·s | ✅ |
| P0-2 | 搜索下架删除 + 启动重建(log-&-continue) | cod·m | ✅ |
| P0-3 | 订单创建主干(keystone) | cod·m | ✅ |
| P0-4 | 短信不泄码 + 频率限制 | cod·s | ✅ |
| P0-5 | Item 输入校验(纯函数 ValidateItem) | cod·s | ✅ |
| P0-6 | P0 测试门 + compose 冒烟 | tester·s | ✅ |
| P1-1 | 订单生命周期 + 押金台账 | cod·l | ✅ |
| P1-2 | 退款流(支付 status 3)+ 取消 | cod·l | ✅ |
| P1-3 | 地理半径检索(lazy 近似) | cod·m | ✅ |
| P1-4 | 测试缝:go test 免基建 | tester·m | ✅ |
| P1-5 | Refresh 轮换(单活跃) | cod·m | ✅ |
| P2-1 | 模型↔Schema 漂移 + 配置 runbook | dev+cod·s | ✅ |
| P2-2 | 密钥 bootstrap + OpenSearch 安全启用 | dev·m | ✅ |
| P2-3 | CI 门(lint/build/vet/test/覆盖率≥15%) | dev·s | ✅ |
| P2-4 | 前端规划书 docs/frontend-plan.md | res/reviewer·s | ✅ |
| P3-1 | README + 本地规划/任务/设计文档 | doc·s | 🟡 |
| P3-2 | 推送规则 + 增量 tag/release 自动化 | dev·s | 🟡 |

## 6. 验收门(持续有效)

```bash
cd server && go build ./... && go vet ./... && go test ./... -count=1   # 全绿(无基建)
docker compose -f deploy/docker-compose.yml up -d                        # 基建一键起
# curl 冒烟: 登录→发物品→公开详情→下单→预付(mock)→回调→订单 status=1
```

## 7. 风险与约束

- **资金路径**:一律条件更新幂等;微信退款需商户证书(CI 仅 mock)。
- **rpm 限流(429)**:并行子代理 ≤3,单代理串行派发。
- **文件≤500 行**;边界输入校验;不提交密钥/.env。