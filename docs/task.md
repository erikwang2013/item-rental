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

## 8. 阶段C — 双端全量重建前端(✅ 本轮完成)

uni-app(`apps/web`)作废删除;并行重建 **Flutter 主App**(`apps/flutter`)+ **微信原生小程序**(`apps/wxapp`),页面映射与契约要点见 [frontend-plan.md §7](frontend-plan.md)。

- [x] **C1 删除 apps/web** — `git rm -r apps/web`(已 staged,随本阶段提交)。
- [x] **C2 Flutter 主App** — models×5 + 13 页 + 主题 #2E7D32 + tabBar;`flutter analyze` 零告警、`flutter test` 3/3、`flutter build web` 成功。
- [x] **C3 wxapp 小程序** — request 封装(信封/401 双 token 刷新单锁)+ api×6 + 13 页四件套;`node --check` 24 JS 全过、JSON 全合法。
- [x] **C4 契约核对(reviewer 独立复核)** — 以 server/ 源码为真源抽查 13 页 API 调用;两端一致,PASS-with-notes。
- [x] **C5 契约偏差处置** — ①前端修正:Flutter `/items` city 参数无效→城市过滤改走 `/items/search`;②文档修正(docs/api.md + frontend-plan.md):`keyword→q`、`size→page_size`、unifiedorder `order_id→order_no+channel`、`/items` 无 city、images 逗号分隔 string、金额为 float 元。

**验证**:`flutter analyze` 零 issue;wxapp `node --check` 全过;`go build && go vet && go test ./...` 后端回归不受前端改动影响。

**已知缺口**:已由阶段D D1/D2 闭环,见 §9。

## 9. 阶段D — 后端缺口补齐 + 双端换真 + profile 鉴权闭环 + 联调冒烟(✅ 本轮完成)

- [x] **D1 按 owner 列物品** — `GET /items?owner_id=`(JWT 本人校验:未登录 401/非本人 403;owner 视图含下架、无 status 过滤);纯函数 `resolveItemOwnerScope` + 4 单测;两端 seller「我的物品」删降级实现换真。
- [x] **D2 头像上传** — `POST /user/avatar`(JWT,multipart file,jpg/jpeg/png/webp ≤4MB),落 `static/uploads/avatars/<uid>_<ts>.<ext>` 直写 user.avatar 返回 `{avatar: URL}`;`services.ValidateAvatarName` 单测;两端 profile 换「选图→上传→显示 URL」。
- [x] **D3 profile 鉴权挂载(残 401 闭环)** — router 补 JWTAuth InsertFilter(此前 profile 路由漏挂,GetUserID 恒空);profile_403_test.go 扩展无 token 401 / 带 token 放行。
- [x] **D4 docs** — api.md(owner_id 语义、POST /user/avatar、notify mock 验签)与本节。
- [x] **D5 真实链路联调冒烟** — 起后端(ITEM_RENTAL_JWT_SECRET + WECHAT_MOCK=1),curl E2E 两轮:首轮打穿 3 个 A 级缺陷,修复后复跑**全断言 PASS**;脚本/断言/复跑命令见 `docs/e2e-smoke-d.md`。
  - 冒烟打穿并修复:A① owner 视图恒 401(GET 公开挂载、JWTAuth 不执行 → List 内 owner_id>0 时显式鉴权);A② WAF `mail_header` 误判 multipart Content-Type → 一切上传 403(扫描副本剥离该头 + `TestMultipartUploadNotBlocked` 回归);A③ notify→MarkPaid 传错标识符(RENT out_trade_no 传给按 ORD order_no 查单)→ 订单永久卡待支付(改经 pay.OrderId 定位订单,移除幂等早退,自愈历史卡单)。
  - 残余观察:登录冷启动后 1-2 分钟窗口内偶发业务 400(绑定读空 body),随后长期稳定,未定位、非本阶段范围,已记入 e2e-smoke-d.md。

**验证**:`go build && go vet && go test ./...` 全绿(新增单测离线);flutter analyze/test/build web 零告警;wxapp node --check 全过;冒烟脚本全断言 PASS。

## 10. 阶段E — 契约修复/钱路闭环/信用分/PII/会话/geo/IPban/CORS/CI/冒烟第二链(✅ 本轮完成)

- [x] **E1 图片契约修复 + 多图上传** — images 统一 JSON 数组串(server 已强制 ≤9,两端原逗号串致带图发布必 400);新 `POST /items/upload`(multipart `files`,≤9×4MB,复用头像校验/落盘/URL 基建);flutter `pickMultiImage`+逐张上传、wxapp `chooseMedia`+串行上传;渲染端 JSON 优先+逗号回退。
- [x] **E2 钱路三缺口** — ①违约押金入物主 `deposit_bal`(原只写台账不动余额);②独立退款仅 status=1 可退、其余 409(原 ≥2 态可被静默退款);③补 return_confirmed/breach/order_cancelled 三类站内信发送点(原仅 payment_success/refunded 两类)。
- [x] **E3 信用分履约公式** — 100 起,按时归还 +5 / 违约 -30 / 已支付后取消 -10,SQL clamp 0-100;新 credit_events 流水表;钩子挂在状态迁移成功后(幂等防重)。
- [x] **E4 PII** — phone 存 sha256 hex(登录等值查询兼容,列扩 64)、real_name AES-GCM 加密(列扩 255,PUT 收 ≤32 字符,GET/登录响应解密);`ITEM_RENTAL_PII_KEY`(64hex)缺失 fail-fast;dev 库 DROP 重建(PII 列变)。
- [x] **E5 refresh 多端会话** — Redis hash 单值改 per-jti 字段集(多端并存);logout 带 refresh_token 仅撤销该端、缺省撤销全部;RotateRefresh 消费 presented jti 防重放。
- [x] **E6 geo 精确化** — 索引加 `location` geo_point 字段 + EnsureGeoIndex(mapping 幂等重建);opensearch 驱动走 WhereGeoDistance 前置过滤(Total 真值),非 ES 驱动保留 Haversine 兜底。
- [x] **E7 IPban 文件持久化** — `storage.NewMemory()` → `NewFile(ipban_file 默认 data/ipban.json)`,30s 自动落盘,重启不丢。
- [x] **E8 CORS + flutter web 联调就绪** — cors 过滤器注册于 SecurityFilter 前(OPTIONS 预检短路);`cors_allow_origins` 默认 *;冒烟验证 GET 带 Origin 回 ACAO 头、OPTIONS 200。
- [x] **E9 CI 前端门 + nightly** — ci.yml 加 flutter-gate job(analyze/test/build web)+ wxapp node --check/JSON 校验;nightly.yml(cron + workflow_dispatch):mysql:8 service + 起服 + `scripts/e2e-smoke.sh`。
- [x] **E10 冒烟脚本化 + 全链复跑** — `scripts/e2e-smoke.sh`(可执行真源)链 1+链 2a(cancel)+链 2b(breach)全 PASS;**A④ 修复**:AdjustDepositBal/AdjustCredit 的 ORM Update 传 SQL 表达式字符串被当绑定参数 → 自阶段B 起押金/信用分从未落库(状态断言掩盖),改 `o.Raw` 三处后链 2 断言钉死(物主入账/credit 100→60 精确落库);冷启动 20 连发登录 0 失败(CopyBody 兜底生效);CORS 冒烟过;root 密码还原。

**验证**:`go build && go vet && go test ./...` 9 包全绿;flutter analyze 零告警/test/build web;wxapp node --check 全过;`bash scripts/e2e-smoke.sh` 链 1+2 全断言 PASS(含退款 409、押金入物主账、credit 精确值、消息×N)。

**已知边界(记录在案)**:信用/通知/台账与状态迁移非同一事务(与既有押金写法一致,故障下可经 credit_events/deposits 台账人工对账);EnsureGeoIndex 每次启动删索引重建(单实例可接受,多实例需查 mapping 存在性);dev 库重建后旧明文 phone 会注册为新账号(上线侧迁移脚本为范围外待办)。