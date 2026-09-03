# API 接口文档

> Base URL: `/api/v1`。除标注「公开」外均需 `Authorization: Bearer <AccessToken>`。
> 响应信封:`{"code":0,"msg":"ok","data":...}`;失败:`{"code":401|403|404|422,"msg":"..."}`。

## 1. 认证 / 用户

| 方法 | 路径 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| POST | /auth/sms | 公开 | 发送验证码(60s 限频,真实模式 Redis 限频;不返回明文验证码) |
| POST | /auth/login | 公开 | 验证码登录(自动注册),返回 access + refresh token |
| POST | /auth/refresh | 公开 | 双 Token 轮换(单活跃 refresh) |
| POST | /auth/logout | JWT | 登出:当前用户 refresh 会话失效 |
| GET | /user/profile | JWT | 获取用户资料 |
| PUT | /user/profile | JWT | 更新用户资料 |
| POST | /user/avatar | JWT | 头像上传(multipart 字段 file,jpg/jpeg/png/webp ≤4MB),直落库返回 {avatar: URL} |

### POST /auth/sms

请求体:`{"phone":"13800138000"}`
成功:`{code:0, msg:"ok"}` — **不含验证码明文**。

### POST /auth/login

请求体:`{"phone":"13800138000","code":"123456"}`
成功:
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "access_token":  "...",
    "refresh_token": "...",
    "access_expires_in": 7200,
    "refresh_expires_in": 604800
  }
}
```

### POST /auth/refresh

请求体:`{"refresh_token":"..."}`
成功:返回新的 access_token + refresh_token(旧 refresh 失效)。

### GET/PUT /user/profile

PUT 请求体可选:`nickname | avatar | real_name | phone`。

### POST /user/avatar

`multipart/form-data`,字段 `file`;扩展名白名单 jpg/jpeg/png/webp,≤4MB。成功后**直接更新 user.avatar** 并返回 `{code:0, data:{avatar: "http://<host>/static/uploads/avatars/<uid>_<ts>.<ext>"}}`,无需再走 PUT profile 传头像。

## 2. 类目

| 方法 | 路径 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| GET | /categories | 公开 | 类目列表 |

## 3. 物品

| 方法 | 路径 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| GET | /items | 公开 | 分页列表,可带 category_id(**不支持 city**,城市过滤走 /items/search);带 owner_id 为「我的物品」视图(JWT 本人,含下架) |
| GET | /items/search | 公开 | 关键词搜索(q)+ 城市/价格过滤 + 半径检索 |
| GET | /items/:id | 公开 | 物品详情 |
| POST | /items | JWT | 发布物品 |
| PUT | /items/:id | JWT(owner) | 修改物品 |
| POST | /items/:id/offshelf | JWT(owner) | 下架(同步删除搜索索引) |

### GET /items 查询参数

`page=1&page_size=20&category_id=1`(分页参数为 page_size;GET /items 无 city 过滤)
`owner_id=<uid>`:「我的物品」视图 — 需 JWT 且 owner_id 等于本人 uid(未登录 401、非本人 403),**包含下架物品**(无 status 过滤),分页/品类过滤同公开列表;不带 owner_id 行为不变(公开仅上架)。

### GET /items/search 查询参数

`q=相机&city=上海&category_id=1&min_price=10&max_price=100&order_by=&page=1&page_size=20` — 关键词参数为 `q`;另支持 `lat&lng&radius_km` 半径检索(边界盒 + Go Haversine 精滤)。

### POST /items 请求体

```json
{
  "title": "单反相机",
  "category_id": 1,
  "daily_price": 30,
  "deposit": 200,
  "stock": 1,
  "city": "上海",
  "lat": 31.2304,
  "lng": 121.4737,
  "images": "https://x.jpg"
}
```

校验规则:Title≤128、Deposit≥0、Stock∈[1,999]、lat∈[-90,90]、lng∈[-180,180]、CategoryId 须存在、owner≠空。

## 4. 订单(全部 JWT)

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | /orders | 创建订单(status=0 待支付) |
| GET | /orders | 我的订单 |
| GET | /orders/:id | 订单详情(renter/owner 可见) |
| POST | /orders/:id/pickup | 取货(1→2,冻结押金) |
| POST | /orders/:id/return_request | 申请归还(2→3) |
| POST | /orders/:id/return_confirm | 确认归还(3→4,解冻+结算) |
| POST | /orders/:id/breach | 违约(3→6,扣押金) |
| POST | /orders/:id/cancel | 取消(0→5,退租金) |

### POST /orders 请求体

```json
{
  "item_id": 1,
  "start_date": "2026-09-03",
  "end_date": "2026-09-06"
}
```

按天计价:days = end-start,RentAmount = days × daily_price,Deposit 取自物品。renter≠owner、item 须上架且 stock>0。

### 7 态流转

```
0 待支付 → 1 待取 → 2 租赁中 → 3 待归还 → 4 已归还(终态)
  │                       │
  └→ 5 已取消(0→5,退款)   └→ 6 违约(3→6,扣押金)
```

全部迁移 = 条件更新幂等;并发回调 / 重复退款安全。

## 5. 支付

| 方法 | 路径 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| POST | /pay/unifiedorder | JWT | 微信预付(JSAPI),mock 模式返回测试 code_url |
| POST | /pay/refund | JWT(owner/renter) | 退款 |
| POST | /pay/notify | 回调验签 | 支付/退款结果回调 |

### POST /pay/unifiedorder 请求体

`{"order_no":"ORD...","channel":"native"}`(order_no 为订单号字符串,channel=native/jsapi)— 成功返回微信 JSAPI pay_params;mock 模式客户端轮询订单状态。

### POST /pay/refund 请求体

`{"order_id":1,"refund_amount":30}`。real 模式需商户双证书,mock 模式短路通过。

### POST /pay/notify

微信支付回调。验签 → 解析 trade_state/refund_status → 订单迁移(已付 / 已退款)。XML 响应:`<xml><return_code><![CDATA[SUCCESS]]></return_code></xml>`。

mock/dev 模式(`WECHAT_MOCK=1`,商户密钥为空串):回调仍需按 V2 规则签名(HMAC-SHA256,参数按字典序拼 `k=v&...&key=`),字段需 return_code/result_code=SUCCESS、out_trade_no(支付单号)、非空 transaction_id、total_fee(分,须等于支付金额)。

## 6. 站内消息(全部 JWT)

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | /messages?unread=1&page=1&page_size=20 | 消息列表(最新优先,支持未读过滤 + 分页),`data.unread` 为未读总数 |
| POST | /messages/:id/read | 标记已读(仅本人消息,403) |

消息类型 `type`:payment_success / payment_refunded / return_confirmed / breach / order_cancelled。由服务端在支付成功回调和退自动写入,无主动推送(前端轮询/下拉刷新感知)。

## 7. 其他公开端点

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | /health | 健康检查,返回 `{"code":0,"msg":"ok"}` |
| GET | /static/mascot.svg | 吉祥物 SVG |

## 8. 错误码

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | /health | 健康检查,返回 `{"code":0,"msg":"ok"}` |
| GET | /static/mascot.svg | 吉祥物 SVG |

## 8. 错误码

| code | 含义 |
| --- | --- |
| 401 | 未登录 / token 无效 / token 已轮换失效 |
| 403 | 无权限(非 owner / 已被封禁 / 触发攻击检测) |
| 404 | 资源不存在 |
| 422 | 请求参数校验失败(金额/坐标/长度/类别存在性) |
| 500 | 服务端错误 |

## 9. 冒烟示例

```bash
# 登录得 token
curl -s -X POST http://127.0.0.1:8080/api/v1/auth/sms \
  -H 'Content-Type: application/json' -d '{"phone":"13800138000"}'
TOKEN=$(curl -s -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"phone":"13800138000","code":"123456"}' | python3 -c "import json,sys; print(json.load(sys.stdin)['data']['access_token'])")

# 发布物品
curl -s -X POST http://127.0.0.1:8080/api/v1/items -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"单反相机","category_id":1,"daily_price":30,"deposit":200,"stock":1}'

# 公开读详情
curl -s http://127.0.0.1:8080/api/v1/items/1
```
