# 前端实施计划（Flutter / 微信小程序）

> 依据后端现状（`server/routers/router.go`、`server/controllers/`、`server/services/`、`server/models/`）编写。
> 覆盖：页面地图、API 依赖、鉴权与令牌、状态机对照、契约要点、未实现/裁剪。

**实现落地**：`apps/flutter`(Flutter 主App)与 `apps/wxapp`(微信原生小程序)为两个平行实现，页面映射见 [§7](#7-实现落地)。

---

## 1. 页面地图

| 页面 | 路径 | 用途 | 核心操作 |
|------|------|------|----------|
| 认证登录 | `/pages/auth/login` | 手机号登录（未注册自动注册） | 输手机号 → 登录；登出 |
| 首页 | `/pages/home/index` | 分类入口 + 在售物品推荐 | 进列表 / 我的 |
| 物品列表 | `/pages/items/list` | 按品类浏览在售物品 | 品类筛选；分页加载 |
| 物品详情 | `/pages/items/detail?id=` | 查看物品、库存、租金 | 查看；选日期下单入口 |
| 下单确认 | `/pages/order/confirm` | 确认租赁日期与费用明细 | 选开始/结束日期；展示天数/租金；提交创建订单 |
| 支付 | `/pages/order/pay` | 订单支付（mock 轮询） | 统一下单；轮询订单状态 |
| 订单中心 | `/pages/order/list` | 按状态查看订单（租入 / 出租） | 状态 Tab 切换；进入详情 |
| 订单详情 | `/pages/order/detail?id=` | 订单流转 | 按状态触发操作（见 §4） |
| 我的 | `/pages/user/index` | 个人信息、订单/物品入口 | 查看/编辑资料；商家入口；登出 |
| 个人资料 | `/pages/user/profile` | 昵称 / 头像 / 实名编辑 | 查看 / 编辑并保存 |
| 商家物品管理 | `/pages/seller/items` | 管理自己发布的物品 | 列表（全部状态）；发布；下架 |

---

## 2. API 依赖清单

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| POST | `/api/v1/auth/login` | 公开 | 手机号登录（自动注册），body `{phone}` |
| POST | `/api/v1/auth/refresh` | 公开 | 双令牌轮换，body `{refresh_token}` |
| POST | `/api/v1/auth/logout` | 需登录 | 登出/撤销 refresh 会话 |
| GET | `/api/v1/user/profile` | 需登录 | 我的资料（`phone` 不下发） |
| PUT | `/api/v1/user/profile` | 需登录 | 更新昵称/头像/实名 |
| POST | `/api/v1/user/avatar` | 需登录 | 头像上传 |
| GET | `/api/v1/categories` | 公开 | 品类列表 |
| GET | `/api/v1/items` | 公开 | 在售物品分页列表（`status=1`）；`owner_id` 为我的物品视图 |
| GET | `/api/v1/items/:id` | 公开 | 物品详情（不过滤状态） |
| POST | `/api/v1/items` | 需登录 | 发布物品 |
| POST | `/api/v1/items/upload` | 需登录 | 物品多图上传 |
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

### 页面 → API 映射

| 页面 | 依赖 API |
|------|----------|
| 认证登录 | `auth/login` |
| 首页 | `categories`、`items` |
| 物品列表 | `items` |
| 物品详情 | `items/:id` |
| 下单确认 | `items/:id`、`orders`（POST） |
| 支付 | `pay/unifiedorder`、`orders/:id`（轮询） |
| 订单中心 | `orders`（GET） |
| 订单详情 | `orders/:id`、`pickup`、`return_request`、`return_confirm`、`breach`、`cancel`、`pay/refund` |
| 我的 | `user/profile`（GET） |
| 个人资料 | `user/profile`（GET/PUT）、`user/avatar` |
| 商家物品管理 | `items`（GET/POST）、`items/upload`、`items/:id`（GET/PUT）、`offshelf` |
| 全局 | `auth/refresh`、`auth/logout`（令牌刷新/登出） |

---

## 3. 鉴权与令牌策略

- **双令牌**：`access`（HS256，TTL 7200s≈2h，`typ=access`）+ `refresh`（TTL 604800s≈7d，`typ=refresh`）。
- **存储**：小程序 `wx.setStorageSync` / Flutter 本地存储；勿入 Cookie。
- **请求注入**：`Authorization: Bearer <access>`；`/orders*`、`/pay*`、`/user/profile`、`/user/avatar` 由 `JWTAuth` 中间件强制校验（失败 HTTP 401）。
- **刷新轮换**：`POST /auth/refresh`（body `{refresh_token}`）→ 返回新双令牌；旧 refresh 立即失效（多端会话并存，单端轮换）。
- **401 处理流程**：
  1. 仅当收到**真实 HTTP 401** 且请求携带 Bearer 时判定 access 过期。
  2. **单飞去重**：并发多个 401 时共享同一个 refresh Promise。
  3. refresh 成功 → 更新双令牌、重放被拒请求（限一次）。
  4. refresh 失败 / 也过期 → 清除令牌、跳转认证登录页。
- **登出**：调 `POST /auth/logout` 撤销服务端 refresh 会话，再清本地令牌。

---

## 4. 状态机对照

后端订单 7 态：`0待支付 → 1待取(已付租金) → 2租赁中 → 3待归还 → 4已归还`；`5已取消`、`6违约` 为终态。除支付/退款外，状态迁移不涉及资金动作。

| 状态 | 展示文案 | 租客可操作 | 房东可操作 |
|------|----------|-----------|-----------|
| 0 待支付 | 待支付 | 去支付、取消 | 查看 |
| 1 待取 | 待取货（已付租金） | 取货 | 查看 |
| 2 租赁中 | 租赁中 | 申请归还 | 查看 |
| 3 待归还 | 待归还 | 查看 | 确认归还、判定违约 |
| 4 已归还 | 已归还 | 查看 | 查看 |
| 5 已取消 | 已取消 | — | — |
| 6 违约 | 违约 | 查看 | 查看 |

**状态 → 动作映射**（按钮仅按状态 + 身份展示）：

| 动作 | 允许状态 | 身份 | 流转 | 后端要点 |
|------|----------|------|------|----------|
| 去支付 | 0 | 租客 | 0 → 1 | 先 `pay/unifiedorder`，支付成功由回调驱动（前端轮询） |
| 取消 | 0 | 租客/房东 | 0 → 5 | `orders/:id/cancel`；幂等；若已支付则自动退款 |
| 取货 | 1 | 租客 | 1 → 2 | `orders/:id/pickup` |
| 申请归还 | 2 | 租客 | 2 → 3 | `orders/:id/return_request` |
| 确认归还 | 3 | 房东 | 3 → 4 | `orders/:id/return_confirm` |
| 判定违约 | 3 | 房东 | 3 → 6 | `orders/:id/breach` |
| 退款（API） | 1 | 租客/房东 | — | `pay/refund`；仅已支付未取可退 |

---

## 5. 契约要点

### 5.1 统一响应

- 成功：`{code:0, msg:"ok", data:{...}}`。
- 业务失败：`{code:<业务码>, msg:"...", data:null}`。
- **HTTP 状态**：除 JWT 中间件（401）外，控制器业务失败不设置 HTTP 状态，**一律 HTTP 200**，错误以 body `code` 为准。前端**必须**以 body `code` 为判断依据。

### 5.2 错误码约定

| code | 含义 | 触发示例 |
|------|------|----------|
| 0 | 成功 | — |
| 400 | 参数错误 | 手机号格式、日期非法、图片格式 |
| 401 | 未认证/令牌失效 | 未登录、token 无效/过期 |
| 403 | 无权限 | 非物主更新/下架物品、非当事双方看订单详情 |
| 404 | 不存在 | 物品/订单不存在 |
| 409 | 状态冲突 | 状态机非法流转（如对已取消订单取货、租赁中退款） |
| 500 | 服务端错误 | — |

> 中间件层例外：`JWTAuth` 失效返回真实 HTTP 401（body 为 `{code,msg}`）。

### 5.3 分页

- 列表接口统一参数：`page`（默认 1）、`page_size`（默认 20，上限 100）。
- 响应 `data` 内返回 `items` + `total` + `page`；前端据此渲染分页/触底加载。

### 5.4 关键字段说明

| 字段 | 说明 |
|------|------|
| `item.images` | **JSON 数组串**（服务端校验 ≤9 张，如 `"[\"https://x.jpg\"]"`；展示前先 `JSON.parse`，失败回退 `split(',')` 容错）。发布 = 先逐张传 `POST /items/upload`（字段 `files`）收 URL，再拼数组串提交 |
| `item.city` | 城市展示字段（列表/详情） |
| 金额单位 | 浮点元（`daily_price`/`rent_amount` 均为 float 元，**非分**） |
| `order.status` | 订单状态（见 §4） |
| `payment.status` | 0 待支付 / 1 成功 / 2 失败 / 3 已退款 |
| `order.pay_trade_no` | 关联支付单号（取 `payments.out_trade_no`） |
| 订单号格式 | `ORD+时间戳+8位随机`；支付单 `RENT+时间戳+订单ID`；退款单 `REF+时间戳+订单ID` |
| 主键 id | **snowflake 字符串契约**:users/items/orders/payments 的 `id` 及引用域(`owner_id`/`renter_id`/`item_id`/`order_id`/`user_id`/owner·renter 子对象 `id`)在 JSON 中一律为**字符串**(>2^53 超 JS Number 安全整数);客户端不得 `Number()/parseInt()` 强转,比较用字符串等值;请求体 `item_id`/`order_id` 传字符串。`category_id` 等枚举仍为数字。flutter 端 Dart int64 安全,`int.tryParse` 双形态容错 |

---

## 6. 未实现 / 裁剪

| 项 | 现状 | 前端方案 |
|----|------|---------|
| 搜索 / 地理检索 | 开源版已裁剪（商业版保留） | 不提供搜索页；首页/列表仅类目浏览 |
| 押金 / 信用分 | 开源版已裁剪（商业版保留） | 不展示押金/信用分；订单仅租金 |
| 站内消息 | 开源版已裁剪（商业版保留） | 不提供消息中心/未读角标 |
| 短信验证码登录 | 开源版已裁剪（商业版保留） | 登录页仅手机号输入 |
| 安全防护 / IP 封禁 | 开源版已裁剪（商业版保留） | 无前端感知 |
| 优惠券 / 积分 / 钱包充值 | 后端无对应模块 | 不在本期范围 |

---

## 7. 实现落地

11 页双端落地映射。两端与 server 契约偏差全部以 server 源码为准：

| # | 页面 | Flutter(`apps/flutter/lib/pages/`) | wxapp(`apps/wxapp/pages/`) |
|---|------|-----------------------------------|---------------------------|
| 1 | 登录 | auth/login | auth/login |
| 2 | 首页 | home/index | home/index |
| 3 | 物品列表 | items/list | items/list |
| 4 | 物品详情 | items/detail | items/detail |
| 5 | 下单确认 | order/confirm | order/confirm |
| 6 | 支付(mock 轮询) | order/pay | order/pay |
| 7 | 订单列表(7 态 tab) | order/list | order/list |
| 8 | 订单详情(流转 5 操作) | order/detail | order/detail |
| 9 | 我的(资料/物品/登出) | user/index | user/index |
| 10 | 编辑资料 | user/profile | user/profile |
| 11 | 我的物品(发布/下架) | seller/items | seller/items |
