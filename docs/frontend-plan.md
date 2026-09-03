# 前端实施计划（小程序 / H5）

> 依据后端现状（`server/routers/router.go`、`server/controllers/`、`server/services/`、`server/middleware/`、`server/models/`）编写。
> 覆盖：页面地图、API 依赖、鉴权与令牌、状态机对照、契约要点、未实现/待办。

**实现落地（阶段C，2026-09）**：原规划为 uni-app(`apps/web`,已删除作废)。现按 §7 落地为两个平行实现——`apps/flutter`(Flutter 主App)与 `apps/wxapp`(微信原生小程序),页面路径映射见 [§7](#7-实现落地阶段c),页面功能与 API 调用以两处实现为基准。

---

## 1. 页面地图

路径为建议值（小程序 / H5 通用 `pages/xxx` 风格）。

| 页面 | 建议路径 | 用途 | 核心操作 |
|------|----------|------|----------|
| 认证登录 | `/pages/auth/login` | 手机号 + 验证码登录（未注册自动注册） | 输手机号 → 发送验证码 → 输码 → 登录；登出（清令牌） |
| 首页 | `/pages/home/index` | 分类入口 + 在售物品推荐 | 进列表 / 搜索 / 我的 / 商家中心 |
| 物品列表 | `/pages/items/list` | 按品类浏览在售物品 | 品类筛选；分页加载；价格排序 |
| 搜索 | `/pages/items/search` | 关键字 + 价格 + 排序 + 地理搜索 | 输关键字；设价格区间；按距离筛选；排序 |
| 物品详情 | `/pages/items/detail?id=` | 查看物品、库存、租金/押金 | 查看；选日期下单入口 |
| 下单确认 | `/pages/order/confirm` | 确认租赁日期与费用明细 | 选开始/结束日期；展示天数/租金/押金；提交创建订单 |
| 支付 | `/pages/order/pay` | 订单支付（native 二维码 / jsapi 调起） | 统一下单；展示二维码或调起微信支付；轮询订单状态 |
| 订单中心 | `/pages/order/list` | 按状态查看订单（租入 / 出租） | 状态 Tab 切换；进入详情 |
| 订单详情 | `/pages/order/detail?id=` | 订单流转与押金状态 | 按状态触发操作（见 §4） |
| 我的 | `/pages/user/index` | 个人信息、押金账户、信用分 | 查看/编辑资料；押金余额；信用分展示（占位）；商家入口 |
| 个人资料 | `/pages/user/profile` | 昵称 / 头像编辑 | 查看 / 编辑并保存 |
| 商家物品管理 | `/pages/seller/items` | 管理自己发布的物品 | 列表（全部状态）；发布；编辑；下架 |

---

## 2. API 依赖清单

### 2.1 端点总表

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| POST | `/api/v1/auth/sms` | 公开 | 发送验证码，body `{phone}` |
| POST | `/api/v1/auth/login` | 公开 | 验证码登录（自动注册），body `{phone,code}` |
| POST | `/api/v1/auth/refresh` | 公开 | 双令牌轮换，body `{refresh_token}` |
| GET | `/api/v1/user/profile` | 需登录* | 我的资料（`phone` 不下发） |
| PUT | `/api/v1/user/profile` | 需登录* | 更新昵称/头像，body `{nickname,avatar}` |
| GET | `/api/v1/categories` | 公开 | 品类列表 |
| GET | `/api/v1/items` | 公开 | 在售物品分页列表（`status=1`） |
| GET | `/api/v1/items/search` | 公开 | 搜索（关键字/价格/排序/地理） |
| GET | `/api/v1/items/:id` | 公开 | 物品详情（不过滤状态） |
| POST | `/api/v1/items` | 需登录 | 发布物品 |
| PUT | `/api/v1/items/:id` | 需登录 | 更新物品（仅物主，403） |
| POST | `/api/v1/items/:id/offshelf` | 需登录 | 下架（仅物主，403） |
| POST | `/api/v1/orders` | 需登录 | 创建订单，body `{item_id,start_date,end_date}` |
| GET | `/api/v1/orders` | 需登录 | 订单列表（租客/房东，分页+状态过滤） |
| GET | `/api/v1/orders/:id` | 需登录 | 订单详情（仅当事双方，403） |
| POST | `/api/v1/orders/:id/pickup` | 需登录 | 取货（租客，1→2） |
| POST | `/api/v1/orders/:id/return_request` | 需登录 | 申请归还（租客，2→3） |
| POST | `/api/v1/orders/:id/return_confirm` | 需登录 | 确认归还（房东，3→4） |
| POST | `/api/v1/orders/:id/breach` | 需登录 | 判定违约（房东，3→6） |
| POST | `/api/v1/orders/:id/cancel` | 需登录 | 取消（0→5，幂等；自动退款） |
| POST | `/api/v1/pay/unifiedorder` | 需登录 | 统一下单，body `{order_no,channel,openid}` |
| POST | `/api/v1/pay/refund` | 需登录 | 发起退款，body `{order_id,refund_reason}` |
| POST | `/api/v1/pay/notify` | 公开 | 微信支付回调（XML，**前端不调用**） |

> \* 用户资料等接口未挂 JWT 中间件，由控制器自行校验（未登录返回 body `code=401`，HTTP 仍为 200）；`/orders*`、`/pay*` 挂了 `JWTAuth` 中间件，鉴权失败返回真实 HTTP 401。

### 2.2 页面 → API 映射

| 页面 | 依赖 API |
|------|----------|
| 认证登录 | `auth/sms`、`auth/login` |
| 首页 | `categories`、`items`、`items/search` |
| 物品列表 | `categories`、`items` |
| 搜索 | `items/search` |
| 物品详情 | `items/:id` |
| 下单确认 | `items/:id`、`orders`（POST） |
| 支付 | `pay/unifiedorder`、`orders/:id`（轮询） |
| 订单中心 | `orders`（GET） |
| 订单详情 | `orders/:id`、`pickup`、`return_request`、`return_confirm`、`breach`、`cancel`、`pay/refund` |
| 我的 | `user/profile`（GET） |
| 个人资料 | `user/profile`（GET/PUT） |
| 商家物品管理 | `items`（GET/POST）、`items/:id`（GET/PUT）、`offshelf` |
| 全局 | `auth/refresh`（令牌刷新） |

（13 个页面，共 30 处依赖引用）

---

## 3. 鉴权与令牌策略

- **双令牌**：`access`（HS256，TTL 7200s≈2h，`typ=access`）+ `refresh`（TTL 604800s≈7d，`typ=refresh`）。
- **存储**：小程序 `wx.setStorageSync` / H5 `localStorage`；仅内存态，勿入 Cookie（防 XSS 泄露）。
- **请求注入**：`Authorization: Bearer <access>`；`/orders*`、`/pay*` 由 `JWTAuth` 中间件强制校验（失败 HTTP 401）。
- **刷新轮换**：`POST /auth/refresh`（body `{refresh_token}`）→ 返回新双令牌；**旧 refresh 立即失效**（单活跃会话，Redis 覆盖），每个 refresh 仅能使用一次。
- **401 处理流程**：
  1. 仅当收到**真实 HTTP 401** 且请求携带 Bearer 时判定 access 过期。
  2. **单飞去重**：并发多个 401 时共享同一个 refresh Promise（进行中不重复发起）。
  3. refresh 成功 → 更新双令牌、重放被拒请求（限一次）。
  4. refresh 失败 / 也过期 → 清除令牌、跳转认证登录页。
  - ⚠️ 登录、发码等公开接口返回的 body `code=401`（如验证码错误）属**业务失败**，不触发刷新。
- **登出**：本地清除双令牌；后端暂**无登出接口**（见 §6）。

---

## 4. 状态机对照

后端订单 7 态：`0待支付 → 1待取(已付租金) → 2租赁中(押金冻结) → 3待归还 → 4已归还(结算)`；`5已取消`、`6违约` 为终态。

| 状态 | 展示文案 | 租客可操作 | 房东可操作 | 支付 / 押金 |
|------|----------|-----------|-----------|-------------|
| 0 待支付 | 待支付 | 去支付、取消 | 查看 | 支付单待支付；押金未动 |
| 1 待取 | 待取货（已付租金） | 取货 | 查看 | 支付单成功；押金未冻结 |
| 2 租赁中 | 租赁中 | 申请归还 | 查看 | 押金冻结（租客余额已扣） |
| 3 待归还 | 待归还 | 查看 | 确认归还、判定违约 | 押金仍冻结 |
| 4 已归还 | 已归还（已结算） | 查看 | 查看 | 押金解冻（余额已返还） |
| 5 已取消 | 已取消 | — | — | 若已支付则退款（支付单已退款） |
| 6 违约 | 违约 | 查看 | 查看 | 押金扣款（不返还） |

**状态 → 动作映射**（按钮仅按状态 + 身份展示）：

| 动作 | 允许状态 | 身份 | 流转 | 后端要点 |
|------|----------|------|------|----------|
| 去支付 | 0 | 租客 | 0 → 1 | 先 `pay/unifiedorder` 拿二维码/参数，支付成功由回调驱动（前端轮询） |
| 取消 | 0 | 租客 | 0 → 5 | `orders/:id/cancel`；幂等；若已支付则自动退款 |
| 取货 | 1 | 租客 | 1 → 2 | `orders/:id/pickup`；押金冻结、余额扣款 |
| 申请归还 | 2 | 租客 | 2 → 3 | `orders/:id/return_request` |
| 确认归还 | 3 | 房东 | 3 → 4 | `orders/:id/return_confirm`；押金解冻、余额返还 |
| 判定违约 | 3 | 房东 | 3 → 6 | `orders/:id/breach`；押金扣款（不返还） |
| 申请退款 | 5/6（可退时） | 租客 | — | `pay/refund`；仅当有成功支付单且未退款时可发起 |

**展示层押金/退款状态推导**（后端无独立字段，需按支付单状态 + 订单状态推导）：

| 场景 | 推导逻辑 | 前端展示 |
|------|----------|----------|
| 押金冻结 | 订单=2/3 且支付单=1 | 「押金已冻结 ¥X」 |
| 押金已返 | 订单=4 且支付单=1 | 「押金已退还 ¥X」 |
| 押金扣除 | 订单=6 且支付单=1 | 「押金扣除 ¥X（违约）」 |
| 已退款 | 支付单=3（已退款） | 「已退款 ¥Y」 |

---

## 5. 契约要点

### 5.1 统一响应

- 成功：`{code:0, msg:"ok", data:{...}}`。
- 业务失败：`{code:<业务码>, msg:"...", data:null}`。
- **HTTP 状态**：除 JWT 中间件（401）与安全过滤（403）外，控制器业务失败不设置 HTTP 状态，**一律 HTTP 200**，错误以 body `code` 为准。前端**必须**以 body `code` 为判断依据，不能只看 HTTP 状态。

### 5.2 错误码约定

| code | 含义 | 触发示例 |
|------|------|----------|
| 0 | 成功 | — |
| 400 | 参数错误 | 手机号格式、日期非法、无权限（未登录的 profile/items 写操作） |
| 401 | 未认证/令牌失效 | 验证码错误；未挂中间件的接口未登录 |
| 403 | 无权限 | 非物主更新/下架物品、非当事双方看订单详情 |
| 404 | 不存在 | 物品/订单不存在 |
| 409 | 状态冲突 | 状态机非法流转（如对已取消订单取货） |
| 429 | 限流 | 验证码发送过于频繁（60s） |

> 中间件层例外：`JWTAuth` 失效返回真实 HTTP 401；安全过滤拦截返回真实 HTTP 403（两者 body 均为 `{code,msg}` 风格）。前端对这两种情况单独处理。

### 5.3 分页

- 列表接口统一参数：`page`（默认 1）、`page_size`（默认 20，上限 100）。
- 响应 `data` 内返回 `items` + `total` + `page`（`/items` 与 `/items/search` 同构）；前端据此渲染分页/触底加载。

### 5.4 搜索参数（`GET /api/v1/items/search`）

| 参数 | 类型 | 说明 |
|------|------|------|
| `q` | string | 关键字（标题/描述）——**注意是 `q`，不是 `keyword`** |
| `category_id` | int | 品类过滤 |
| `min_price` / `max_price` | float | 日租金区间 |
| `order_by` | string | 排序：空（默认最新）/ `price_asc` / `price_desc` |
| `lat` / `lng` / `radius_km` | float | 地理搜索：三者同时传才生效；Haversine 半径过滤，**缺坐标的物品会被跳过** |
| `city` | string | 城市精确过滤（`/items/search` 已接线；`/items` 不支持 city） |
| `page` / `page_size` | int | 分页 |

> ⚠️ 地理过滤为惰性后置过滤：命中总数可能小于返回条数，分页体验需接受「某页不满」；当搜索降级（引擎未配置）时地理过滤不生效，前端需兜底提示。

### 5.5 关键字段说明

| 字段 | 说明 |
|------|------|
| `user.deposit_bal` | 押金余额（冻结/解冻/扣款后实时反映） |
| `user.credit_score` | 信用分（默认 100；见 §6：当前无变动来源） |
| `item.images` | **JSON 数组串**（服务端校验 ≤9 张，如 `"[\"https://x.jpg\"]"`；展示前先 `JSON.parse`，失败回退 `split(',')` 容错）。发布 = 先逐张传 `POST /items/upload`（字段 `files`）收 URL，再拼数组串提交 |
| `item.city` / `lat` / `lng` | 地理信息（`city` 参与 `/items/search` 精确过滤；`/items` 不读 city） |
| 金额单位 | 浮点元（`daily_price`/`deposit`/`rent_amount` 均为 float 元，**非分**） |
| `order.status` | 订单状态（见 §4） |
| `payment.status` | 0 待支付 / 1 成功 / 2 失败 / 3 已退款 |
| `order.pay_trade_no` | 关联支付单号（取 `payments.out_trade_no`） |
| 订单号格式 | `ORD+时间戳+8位随机`；支付单 `RENT+时间戳+订单ID`；退款单 `REF+时间戳+订单ID` |

---

## 6. 未实现 / 待办

以下内容为**本期已实现 / 已裁剪**清单（阶段B 已补齐后端项），前端按需接线：

| 项 | 现状 | 前端方案 |
|----|------|---------|
| 登出接口 | ✅ **已实现** `POST /auth/logout`（JWT，使 refresh 会话失效） | 登出 = 清本地令牌 + 调 logout；调 `auth/refresh` 前若已登出会返回失败 |
| 信用分展示源 | `credit_score` 仅默认 100，无变动逻辑 | 「我的」页**占位展示**（读 `user/profile` 字段），标注「敬请期待」 |
| 信用分变动 / 违约扣分 | 违约仅扣押金，无扣分 | 占位；后端定稿后接入 |
| PII 加密下发 | 手机号、实名、身份证加密存储，`phone` 不下发 | 不做明文展示；如需展示需后端新接口 |
| 退款通知 | ✅ **已实现**：退款成功写入站内消息（`type=payment_refunded`） | 通过消息中心 / 订单轮询感知 |
| 支付成功回调后即时态 | ✅ **已实现**：回调异步写入站内消息（`type=payment_success`） | 支付页轮询 `orders/:id` + 消息中心确认 |
| `city` 搜索参数 | ✅ **已实现**：`GET /items/search?city=北京` 已接线（OpenSearch 精确过滤） | 搜索页可提供城市维度筛选 |
| 消息 / 通知中心 | ✅ **已实现**：`messages` 表 + `GET /messages` + `POST /messages/:id/read` + 未读数 | 前端可做消息入口 + 未读角标 |
| 优惠券 / 积分 / 钱包充值 | 后端无对应模块 | 不在本期范围 |

---

## 7. 实现落地（阶段C）

13 页（+ 消息中心）双端落地映射。两端与 server 契约偏差全部以 server 源码为准（见 task.md 阶段C 记录）：

| # | 页面 | Flutter(`apps/flutter/lib/pages/`) | wxapp(`apps/wxapp/pages/`) |
|---|------|-----------------------------------|---------------------------|
| 1 | 登录 | auth/login | auth/login |
| 2 | 首页 | home/index | home/index |
| 3 | 物品列表 | items/list | items/list |
| 4 | 搜索(q+city) | items/search | items/search |
| 5 | 物品详情 | items/detail | items/detail |
| 6 | 下单确认 | order/confirm | order/confirm |
| 7 | 支付(mock 轮询) | order/pay | order/pay |
| 8 | 订单列表(7 态 tab) | order/list | order/list |
| 9 | 订单详情(流转 5 操作) | order/detail | order/detail |
| 10 | 我的(资料/物品/登出) | user/index | user/index |
| 11 | 编辑资料 | user/profile | user/profile |
| 12 | 我的物品(发布/下架) | seller/items | seller/items |
| 13 | 消息中心(角标/已读) | messages/index | messages/index |

> 已知缺口：无 `GET /items?owner_id=` 端点 → seller「我的物品」两端均为降级实现（wxapp 本地记录发布 id；flutter 列全部上架物品靠 403 拦截），补该端点后两端可换真列表。
