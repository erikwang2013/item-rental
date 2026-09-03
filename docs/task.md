# 项目任务 — 多端租赁平台 item-rental

> 本地任务文档(仓库内长期保留)。规划见 [plan.md](plan.md),技术决策见 [design.md](design.md)。

状态图例:🟡 进行中 / ⏳ 待办 / ✅ 已完成 / ➖ 已裁剪(out-of-scope 或在别处处理)。

## 1. 任务总览

| 阶段 | 总任务 | 已完成 | 进行中 | 待办 |
| --- | --- | --- | --- | --- |
| P0 正确性门 | 6 | 6 | 0 | 0 |
| P1 领域完备 | 5 | 5 | 0 | 0 |
| P2 加固 | 4 | 4 | 0 | 0 |
| P3 收尾 | 6 | 6 | 0 | 0 |

## 2. P0 — 正确性门(✅ 全部完成)

- [x] **P0-1** 路由鉴权纠正 — router.go 方法感知过滤器;GET `/items/:id` 公开,POST/PUT/offshelf 需 JWT。
- [x] **P0-2** 搜索下架删除 + 启动重建 — RemoveItem 换 UnsearchableSync;启动 ReindexAll,log-&-continue。
- [x] **P0-3** 订单创建主干 — controllers/order_controller.go + services/order.go(OrderNo/days×price/押金/起止日/renter≠owner/上架校验);POST+GET orders。
- [x] **P0-4** 短信不再泄码 + 频率限制 — 响应无明文验证码;手机号 11 位校验;real 模式 Redis 限频。
- [x] **P0-5** Item 输入校验 — 纯函数 ValidateItem(CategoryId/Deposit/Stock/lat/lng/Title/Images/长度)。
- [x] **P0-6** P0 测试门 — 全量套件 + 纯函数单测 + compose 冒烟(build/vet/test 全绿)。

## 3. P1 — 领域完备(✅ 全部完成)

- [x] **P1-1** 订单生命周期 + 押金台账 — 7 态迁移(取货冻结/归还确认解冻/违约扣款),条件更新幂等。
- [x] **P1-2** 退款流 + 取消 — payments/gateway.Refund(mock 短路)、notify 解析 refund_status、MarkRefunded、POST /orders/:id/cancel。
- [x] **P1-3** 地理半径检索 — WhereRange 边界盒 + Go Haversine 精滤;`ponytail:` 注明升级路径。
- [x] **P1-4** 测试缝 — `go test ./...` 免基建(纯函数迁出、search 惰性空安全)。
- [x] **P1-5** Refresh 轮换 — Redis `auth:refresh:{uid}` 单活跃 hash,refresh 轮换双 token。

## 4. P2 — 加固(✅ 全部完成)

- [x] **P2-1** 模型↔Schema 漂移 + 配置 runbook — docs/config-runbook.md 32 键覆盖表。
- [x] **P2-2** 密钥 bootstrap + OpenSearch 安全启用 — prod compose `${VAR:?}` 必设;端口覆盖 3307/16379/9200。
- [x] **P2-3** CI 门 — golangci-lint + build/vet/test + 覆盖率 ≥15%(基线 25.9%)。
- [x] **P2-4** 前端规划书 — docs/frontend-plan.md(仅规划,无代码任务)。

## 5. P3 — 收尾(✅ 本轮全部完成)

- [x] **P3-1** README.md + 本地规划/任务/设计文档(本文件 + plan.md + design.md)。
- [x] **P3-2** 推送规则 — scripts/push-release.sh:获取最新版本→push→按新版本增量建 tag + GitHub Release。
- [x] **P3-3** CI Quality Gate — golangci-lint default(errcheck)7 处全修齐,build/vet/test/coverage/lint 全绿。
- [x] **P3-4** v1.0.0 正式发布 — main 分支 push + tag 创建 + GitHub Release 创建 + Release 指向 CI 通过的 main HEAD(d207f5f)。
- [x] **P3-5** 吉祥物「租租」— docs/mascot.svg + 启动横幅 + GET /static/mascot.svg + README 展示。
- [x] **P3-6** docs/api.md 独立 API 接口文档 + 功能特性图高清化(960×560 → 1200×700)。

## 6. 已裁剪(记录在案)

| 裁剪项 | 原因 | 恢复条件 |
| --- | --- | --- |
| 独立退款-notify 处理器 | 并入现有 notify 分支 | 退款报文与支付报文需区分时 |
| OpenSearch 精确 geo_distance | 边界盒 + Haversine 已够 | 需距离排序/大半径精度时 |
| Refresh 撤销清单 | 单活跃轮换即可 | 多端同时登录需求 |
| golangci-lint 调参 | 用默认配置 | CI 误报变多时 |
| 信用分逻辑 | OUT-OF-SCOPE | 独立立项 |
| PII 加密 | OUT-OF-SCOPE(schema 保留字段) | 合规要求 |
| 支付宝支付 | OUT-OF-SCOPE(config 预留字段已移除,无实现) | 独立接入时 |

## 7. 阶段B — 服务端待办补齐(✅ 全部完成)

对齐 [frontend-plan.md §6](frontend-plan.md)「未实现/待办」中的后端可交付项:

- [x] **B1 登出接口** — `POST /auth/logout`(JWT)+ refresh 会话失效(Redis Del)+ `TestLogoutInvalidatesRefresh`。
- [x] **B2 站内消息中心** — `messages` 表(schema + ORM 模型)+ `GET /messages`(分页/未读过滤)+ `POST /messages/:id/read`(本人校验)+ `services.Send`/`CountUnread`。
- [x] **B3 消息接入生命周期** — 支付成功(回调异步写 `payment_success`)、退款成功(写 `payment_refunded`),不阻断主流程。
- [x] **B4 city 搜索接线** — `GET /items/search?city=` 透传至 OpenSearch 精确过滤(engine 已有 `Where("city", q.City)`)。

**验证**:`go build ./... && go vet ./... && go test ./... -count=1` 全绿,聚合覆盖率 26.0%(CI 门 ≥15%)。