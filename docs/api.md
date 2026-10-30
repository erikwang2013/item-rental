# API 接口文档

> Base URL: `/api/v1`。除标注「公开」外均需 `Authorization: Bearer <AccessToken>`。
> 响应信封:`{"code":0,"msg":"ok","data":...}`;失败:`{"code":400|401|403|404|409|500,"msg":"..."}`。
> **主键说明**:users/items/orders/payments 主键为 **snowflake id**,JSON 中一律以**字符串**输出(如 `"id":"1943…"`),数值超过 JS `Number.MAX_SAFE_INTEGER(2^53)`,客户端必须按字符串处理(不得 Number/parseInt 强转)。引用型字段(owner_id/renter_id/item_id/order_id/user_id 及 owner/renter 子对象 id)同规则;category_id 等枚举仍为数字。下单请求体 `item_id` 传字符串。

## 1. 认证 / 用户

| 方法 | 路径 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| POST | /auth/login | 公开 | 手机号登录(仅 dev 或 `ITEM_RENTAL_LOGIN_MOCK=1`;自动注册),返回 access + refresh token |
| POST | /auth/refresh | 公开 | 双 Token 轮换(多端会话并存,轮换仅使当前端旧 refresh 失效) |
| POST | /auth/logout | JWT | 登出;请求体可选 `refresh_token`(带则仅撤销该端会话,缺省撤销当前用户全部会话) |
| GET | /user/profile | JWT | 获取用户资料 |
| PUT | /user/profile | JWT | 更新昵称/头像/实名 |
| POST | /user/avatar | JWT | 头像上传(multipart 字段 file,jpg/jpeg/png/webp ≤4MB),直落库返回 {avatar: URL} |

### POST /auth/login

请求体:`{"phone":"13800138000"}`(11 位数字且以 1 开头;未注册自动创建账号)。

成功:
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "access_token":  "...",
    "refresh_token": "...",
    "user": {
      "id": "1943…",
      "nickname": "用户8000",
      "avatar": "",
      "real_name": "",
      "status": 1,
      "created_at": "...",
      "updated_at": "..."
    }
  }
}
```

### POST /auth/refresh

请求体:`{"refresh_token":"..."}`
成功:返回新的 access_token + refresh_token(该端旧 refresh 失效,他端会话不受影响)。

### POST /auth/logout

请求体可选:`{"refresh_token":"..."}`。
带 refresh_token(须属于当前用户)→ 仅撤销该端会话;缺省或解析失败 → 撤销该用户全部会话。恒返回成功。

### GET/PUT /user/profile

PUT 请求体可选:`nickname | avatar | real_name`。`real_name` 明文 ≤32 字符,服务端 AES-GCM 加密存储;GET /user/profile 返回解密后的明文。手机号仅以 HMAC-SHA256 哈希存储,不下发。

### POST /user/avatar

`multipart/form-data`,字段 `file`;扩展名白名单 jpg/jpeg/png/webp,≤4MB。成功后**直接更新 user.avatar** 并返回 `{code:0, data:{avatar: "http://<host>/static/uploads/avatars/<uid>_<ts>.<ext>"}}`,无需再走 PUT profile 传头像。

## 2. 类目

| 方法 | 路径 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| GET | /categories | 公开 | 类目列表 |

## 3. 物品

| 方法 | 路径 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| GET | /items | 公开 | 分页列表,可带 category_id;带 owner_id 为「我的物品」视图(JWT 本人,含下架) |
| GET | /items/:id | 公开 | 物品详情 |
| POST | /items | JWT | 发布物品 |
| POST | /items/upload | JWT | 图片上传(multipart 多文件字段 `files`) |
| PUT | /items/:id | JWT(owner) | 修改物品 |
| POST | /items/:id/offshelf | JWT(owner) | 下架 |

### GET /items 查询参数

`page=1&page_size=20&category_id=1`(分页参数为 page_size)
`owner_id=<uid>`:「我的物品」视图 — 需 JWT 且 owner_id 等于本人 uid(未登录 401、非本人 403),**包含下架物品**(无 status 过滤),分页/品类过滤同公开列表;不带 owner_id 行为不变(公开仅上架)。

### GET /items/:id 详情响应富化

详情在物品字段外附带房东公开信息(可选,列表不含):`"owner": {"id": 7, "nickname": "房东", "avatar": "http://..."}`。owner 仅含昵称/头像,**不含 phone/real_name/status 等任何 PII 或私密字段**;房东行缺失(孤儿数据)时该键省略。

### POST /items 请求体

```json
{
  "title": "单反相机",
  "category_id": 1,
  "daily_price": 30,
  "stock": 1,
  "city": "上海",
  "images": "[\"https://host/static/uploads/items/1_171_0.jpg\"]",
  "desc": "9成新"
}
```

`images` 为 **JSON 数组串**(≤9 张;先 POST /items/upload 逐张上传获取 URL,再拼成数组串提交)。

校验规则:Title≤128、DailyPrice>0、Stock∈[1,999]、CategoryId 须存在、owner≠空;images 须为合法 JSON 数组串且 ≤9 张。

### POST /items/upload 请求体

`multipart/form-data`,字段 `files`(可一次多 part 或逐张单 part;每次调用 ≤9 张),每张扩展名白名单 jpg/jpeg/png/webp、≤4MB。
成功返回 `{code:0, data:{urls:["http://<host>/static/uploads/items/<uid>_<ts>_<i>.<ext>", ...]}}`;任一张失败整批不落库返回 400。

## 4. 订单(全部 JWT)

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | /orders | 创建订单(status=0 待支付) |
| GET | /orders | 我的订单 |
| GET | /orders/:id | 订单详情(renter/owner 可见) |
| POST | /orders/:id/pickup | 取货(1→2) |
| POST | /orders/:id/return_request | 申请归还(2→3) |
| POST | /orders/:id/return_confirm | 确认归还(3→4) |
| POST | /orders/:id/breach | 违约(3→6) |
| POST | /orders/:id/cancel | 取消(0→5;已支付则随后自动退款) |

### GET /orders/:id 详情响应富化

详情在订单字段外附带双方公开信息(可选,列表不含):`"owner": {...}`(房东)与 `"renter": {...}`(租客),子对象字段同物品详情 owner 形态(id/nickname/avatar,不含 PII)。仅租客或房东本人可看,客户端按自身角色读取对方。

### POST /orders 请求体

```json
{
  "item_id": "1943…",
  "start_date": "2026-09-03",
  "end_date": "2026-09-06"
}
```

按天计价:days = end-start+1,RentAmount = days × daily_price。renter≠owner、item 须上架且 stock>0。

### 7 态流转

```
0 待支付 → 1 待取 → 2 租赁中 → 3 待归还 → 4 已归还(终态)
  │                       │
  └→ 5 已取消(0→5,退款)   └→ 6 违约(3→6)
```

全部迁移 = 条件更新幂等;并发回调 / 重复退款安全。除支付/退款外,状态迁移不涉及资金动作。

## 5. 支付

| 方法 | 路径 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| POST | /pay/unifiedorder | JWT | 微信预付(JSAPI);mock 模式下单即自动完成支付(订单变已付) |
| POST | /pay/refund | JWT(owner/renter) | 退款 |
| POST | /pay/notify | 回调验签 | 支付/退款结果回调(mock 模式拒绝) |

### POST /pay/unifiedorder 请求体

`{"order_no":"ORD...","channel":"native","openid":""}`(order_no 为订单号字符串,channel=native/jsapi;jsapi 需 openid)— 成功返回微信 JSAPI pay_params;mock 模式下单即自动完成支付,客户端轮询订单状态即可看到已付。

### POST /pay/refund 请求体

`{"order_id":"1943…","refund_reason":"拍错了"}`。real 模式需商户双证书,mock 模式短路通过。
**仅订单 status=1(待取)可退款**,其余状态返回 409「当前订单状态不允许退款」(防止租赁中/已归还订单被静默退款)。

### POST /pay/notify

微信支付回调。验签 → 解析 trade_state/refund_status → 订单迁移(已付 / 已退款)。XML 响应:`<xml><return_code><![CDATA[SUCCESS]]></return_code></xml>`。

mock 模式(`WECHAT_MOCK=1`)下支付已在统一下单内完成,**`/pay/notify` 一律拒绝回调**。真实模式回调需按 V2 规则签名(HMAC-SHA256,参数按字典序拼 `k=v&...&key=`,商户密钥为空时验签直接失败),字段需 return_code/result_code=SUCCESS、out_trade_no(支付单号)、非空 transaction_id、total_fee(分,须等于支付金额)。

## 6. 其他公开端点

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | /health | 健康检查,返回 `{"code":0,"msg":"ok"}` |
| GET | /static/mascot.svg | 吉祥物 SVG |

## 7. 错误码

| code | 含义 |
| --- | --- |
| 400 | 请求参数校验失败(金额/长度/图片/手机号/日期等) |
| 401 | 未登录 / token 无效 / token 已轮换失效 |
| 403 | 无权限(非 owner / 非订单参与者 / 账号禁用) |
| 404 | 资源不存在 |
| 409 | 订单当前状态不允许该操作(如租赁中退款、非待支付取消) |
| 500 | 服务端错误 |

## 8. 冒烟示例

```bash
# 登录得 token
TOKEN=$(curl -s -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"phone":"13800138000"}' | python3 -c "import json,sys; print(json.load(sys.stdin)['data']['access_token'])")

# 发布物品
curl -s -X POST http://127.0.0.1:8080/api/v1/items -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"单反相机","category_id":1,"daily_price":30,"stock":1}'

# 公开读详情
curl -s http://127.0.0.1:8080/api/v1/items/1
```
