# 项目任务 — 多端租赁平台 item-rental

> 当前开源版任务状态。历史阶段记录见 git 提交历史；本文件只描述当前代码。

## 1. 当前范围

- 后端:Go 1.24 + Beego v2.3 + MySQL 8;Redis 可选(refresh 会话)。
- 数据表:users / categories / items / orders / payments(5 张)。
- 控制器:auth / user / category / item / order / payment(6 个),含头像与物品图上传。
- 客户端:Flutter 主App 11 页 + 微信原生小程序 11 页。
- 支付:微信支付网关抽象,mock 默认,真实商户模式预留。

## 2. 已完成

| 模块 | 状态 | 说明 |
| --- | --- | --- |
| 认证与账号 | ✅ | 手机号直接登录(mock 入口,自动注册)、JWT 双 Token、refresh 轮换、单端登出 |
| 用户资料 | ✅ | 昵称/头像/实名;phone HMAC-SHA256、real_name AES-GCM |
| 物品管理 | ✅ | 类目、发布/修改/下架、多图上传、owner 公开信息富化 |
| 订单与租赁 | ✅ | 按天计价、7 态状态机、取货/归还申请/归还确认/违约、双方角色校验 |
| 支付与退款 | ✅ | 统一下单、回调验签、退款守卫(仅待取可退)、取消自动退款 |
| 质量 | ✅ | Go 离线单测、Flutter test、wxapp JS/JSON 校验、CI、E2E 冒烟脚本 |

## 3. 已裁剪(开源版)

| 功能 | 状态 | 说明 |
| --- | --- | --- |
| 押金与信用分 | 已移除 | 商业版保留;开源版订单仅租金 |
| 搜索与地理位置 | 已移除 | 商业版保留 OpenSearch/geo |
| 站内消息 | 已移除 | 商业版保留 |
| 短信验证码登录 | 已移除 | 开源版手机号直接登录(mock);商业版保留短信 |
| security-go / IP 封禁 | 已移除 | 商业版保留 |

## 4. 验证命令

```bash
cd server && go build ./... && go vet ./... && go test ./... -count=1
cd apps/flutter && flutter analyze && flutter test
cd apps/wxapp && find . -name '*.js' -exec node --check {} \;
bash scripts/e2e-smoke.sh   # 需 MySQL;REBUILD_DB=1 时重建 rental 库
```

## 5. 上线前置

| 项 | 要求 |
| --- | --- |
| PII 密钥 | `ITEM_RENTAL_PII_KEY` 64 位 hex |
| JWT 密钥 | `ITEM_RENTAL_JWT_SECRET` 强随机(32+ 字节) |
| 微信支付 | `WECHAT_MOCK=0` + 商户号/密钥/回调地址/证书 |
| Redis | 多实例部署 `redis_enabled=true` |
| CORS | 生产收紧 `cors_allow_origins` |
| 数据库 | 生产关闭 dev 自动建表(`BEEGO_RUNMODE=prod`) |
