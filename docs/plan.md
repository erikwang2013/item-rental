# 项目规划 — 多端租赁平台 item-rental

> 当前开源版规划。历史路线图见 git 提交历史。

## 1. 定位

开源精简版物品租赁后端:房东发布物品,租客浏览、下单、支付租金、取件使用、按约归还;平台以 7 态订单状态机贯穿履约,配套退款与违约处置。

商业版在此之上保留搜索/geo、押金与信用分、站内消息、短信登录、security-go 安全防护等模块;这些模块不在本仓库。

## 2. 当前基线

### 端点

`/health`;auth/login、refresh、logout;user/profile GET+PUT、user/avatar;categories GET;items GET/:id/POST/PUT/offshelf/upload;orders POST/GET/:id/pickup/return_request/return_confirm/breach/cancel;pay/unifiedorder、refund、notify。

### 数据表

users / categories / items / orders / payments(5 张)。

### 客户端

Flutter 主App 11 页 + 微信原生小程序 11 页;页面映射见 [frontend-plan.md](frontend-plan.md)。

### 支付

微信支付网关抽象:mock 默认(预付/回调验签/退款),真实商户模式预留;退款仅 status=1(待取)可退。

## 3. 后续可做

| 项 | 说明 |
| --- | --- |
| 真实微信支付接入 | 商户资质、证书、回调域名就绪后切换 `WECHAT_MOCK=0` |
| 生产部署 | MySQL/Redis 编排、备份、监控、反向代理与 HTTPS |
| PII 合规审计 | 手机号哈希/实名加密已就位,按合规要求补充流程 |
| 商业版功能 | 搜索/geo、押金/信用分、站内消息、短信登录、安全防护仅商业版维护 |

## 4. 验收门

```bash
cd server && go build ./... && go vet ./... && go test ./... -count=1
cd apps/flutter && flutter analyze && flutter test
cd apps/wxapp && find . -name '*.js' -exec node --check {} \;
bash scripts/e2e-smoke.sh
```

## 5. 风险与约束

- **资金路径**:退款状态守卫;notify 回调重复投递安全;退款 real 模式需商户证书。
- **PII 密钥**:缺失或非法长度启动即 panic,部署前必须配置。
- **ID 契约**:snowflake id > 2^53,JSON 一律字符串,客户端不得强转数字。
- **文件≤500 行**;边界输入校验;不提交密钥/.env。
