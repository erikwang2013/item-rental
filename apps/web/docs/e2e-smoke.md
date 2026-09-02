# E2E 冒烟链路

> 一条"发布 → 搜索 → 下单 → 支付(mock) → 取货 → 归还"主链路。
> 后端 `http://127.0.0.1:8080/api/v1`；前端 `pnpm dev` 代理 `/api/v1`。

## 步骤

| # | 动作 | 接口/页面 | 预期 |
|---|------|-----------|------|
| 1 | 发验证码 | POST /auth/sms | code:0,无明文 |
| 2 | 登录 | POST /auth/login | 双令牌 |
| 3 | 发布物品(房东) | POST /items | status 上架,得 id |
| 4 | 搜索 | GET /items/search | list 含该物品 |
| 5 | 创建订单(租客) | POST /orders | status=0 |
| 6 | 统一下单(mock) | POST /pay/unifiedorder | pay_params |
| 7 | 模拟回调 | POST /pay/notify (SUCCESS) | status=1 |
| 8 | 取货 | POST /orders/:id/pickup | status=2,押金冻结 |
| 9 | 申请归还 | POST /orders/:id/return_request | status=3 |
| 10 | 确认归还(房东) | POST /orders/:id/return_confirm | status=4,解冻 |

## 一键脚本(后端直调)

```bash
B=http://127.0.0.1:8080/api/v1
TOK(){ curl -s -X POST $B/auth/login -H 'Content-Type: application/json' \
  -d "{\"phone\":$1,\"code\":\"123456\"}" | python3 -c "import json,sys;print(json.load(sys.stdin)['data']['access_token'])"; }
A=$(TOK 13800138001)   # 房东
B2=$(TOK 13800138002)   # 租客
ID=$(curl -s -X POST $B/items -H "Authorization: Bearer $A" -H 'Content-Type: application/json' \
  -d '{"title":"单反相机","category_id":1,"daily_price":30,"deposit":200,"stock":1}' \
  | python3 -c "import json,sys;print(json.load(sys.stdin)['data']['id'])")
OID=$(curl -s -X POST $B/orders -H "Authorization: Bearer $B2" -H 'Content-Type: application/json' \
  -d "{\"item_id\":$ID,\"start_date\":\"2026-09-03\",\"end_date\":\"2026-09-06\"}" \
  | python3 -c "import json,sys;print(json.load(sys.stdin)['data']['id'])")
curl -s -X POST $B/pay/unifiedorder -H "Authorization: Bearer $B2" \
  -H 'Content-Type: application/json' -d "{\"order_id\":$OID}"
curl -s -X POST $B/pay/notify -d '<xml><return_code>SUCCESS</return_code><trade_state>SUCCESS></xml>'
curl -s -X POST $B/orders/$OID/pickup -H "Authorization: Bearer $B2"
curl -s -X POST $B/orders/$OID/return_request -H "Authorization: Bearer $B2"
curl -s -X POST $B/orders/$OID/return_confirm -H "Authorization: Bearer $A"
```

> mock 回调的 out_trade_no 需匹配后端 payments.out_trade_no(RENT+时间戳+订单ID)。

## 前端页面冒烟

1. `cd apps/web && pnpm install && pnpm dev`
2. H5 打开 → 登录页验证码登录
3. 首页分类/推荐 → 点物品进详情 → 立即租用
4. 选日期 → 确认下单 → 去支付(mock)→ 轮询到"待取货"
5. 订单中心 Tab → 订单详情 取货/申请归还
6. 切房东账号 → 确认归还 → "已归还"
