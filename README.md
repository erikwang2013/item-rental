# 多端租赁平台 · item-rental

> 物品租赁**后端服务** — 搜索、下单、支付、押金与违约的完整闭环。

![Go](https://img.shields.io/badge/Go-1.24.2-00ADD8?logo=go&logoColor=white)
![Beego](https://img.shields.io/badge/Beego-v2.3.10-339AF0)
![License](https://img.shields.io/badge/License-MIT-green)

<p align="center">
  <img src="docs/mascot.svg" alt="吉祥物租租" width="150">
  <br><b>吉祥物 · 租租猫</b> — 🐱 猫会看家守物、叼物回家,天生贴合「出租闲置、按约归还」;穿 indigo 小马甲 + 戴 ¥ 胸牌 + 右手握钥匙,把「平台管家、按时交付」写进造型。服务启动时控制台也会招手(`/static/mascot.svg` 可见)。
</p>

---

## 项目介绍

一个面向**小程序 / H5 / 管理后台**的 P2P 物品租赁平台后端。房东发布闲置物品,按天计价并收取押金;租客搜索、下单、支付租金、取件使用、按约归还;平台以 7 态订单状态机贯穿租赁全流程,配套押金台账与退款/违约处置。

技术底座:Go 1.24 + Beego v2.3(MVC)+ MySQL + Redis + OpenSearch,接入微信支付 V3(JSAPI 预付/回调/退款),并以 security-go 中间件做攻击检测与 IP 封禁。

## 功能特性

<p align="center"><img src="docs/features.svg" alt="功能特性全景" width="900"></p>

- **认证与账号**:手机号验证码登录(mock/真实短信)、JWT 双 Token(Access 2h + Refresh 7d 轮换)、验证码 60s 限频、用户资料/信用分/押金账户。
- **物品与搜索**:类目体系、上下架管理、OpenSearch 全文检索、关键词×城市×类目筛选、地理位置半径检索(Haversine)。
- **安全与风控**:security-go 攻击检测、统一边界输入校验(金额/坐标/长度)、条件更新幂等防重复回调/退款。
- **订单与租赁**:按天计价 + 押金核算、7 态生命周期、取货/归还申请/归还确认/违约、租客与房东双角色权限。
- **支付与退款**:微信支付 V3 预付(JSAPI)、异步回调验签、退款(mock/商户证书)、订单取消自动退款。
- **押金与结算**:deposits 押金台账(冻结/解冻/扣款)、扣押金退租金联动。

## 技术架构

<p align="center"><img src="docs/architecture.svg" alt="系统架构图" width="840"></p>

| 层 | 组件 |
| --- | --- |
| 客户端 | 微信小程序 · H5/Web · 管理后台 |
| 接入层 | Beego Router(方法感知鉴权) · security-go · JWT 双 Token |
| 业务层 | Controllers(统一信封) · Services(纯业务逻辑) · 集成网关(payments/search/models/middleware) |
| 数据与基础设施 | MySQL · Redis · OpenSearch 2.14 · 微信支付商户平台 |

## 订单生命周期

<p align="center"><img src="docs/lifecycle.svg" alt="订单生命周期 · 7 态状态机" width="840"></p>

| 状态 | 含义 | 主要动作 |
| --- | --- | --- |
| 0 待支付 | 仅占位,租金未付 | — |
| 1 待取 | 租金已付 | 0→1 支付回调 |
| 2 租赁中 | 押金冻结 | 1→2 取货 |
| 3 待归还 | 租客申请归还 | 2→3 归还申请 |
| 4 已归还 | 结算,押金解冻 | 3→4 归还确认 |
| 5 已取消 | 退回押金并退款租金 | 0→5 取消(退款) |
| 6 违约 | 扣押金抵偿违约金 | 3→6 违约 |

资金流:0→1 支付租金;1→2 冻结押金;3→4 解冻押金并结算;退款在 0→5 支线处理。

## 项目结构

```
item-rental/
├── server/                  # Go 后端(Beego v2)
│   ├── main.go              # 入口(启动迁移 + 索引重建)
│   ├── conf/app.conf        # 配置(环境变量覆盖见 config-runbook)
│   ├── routers/router.go    # 路由与方法感知鉴权
│   ├── controllers/         # 控制器(统一 OK/Fail 信封)
│   ├── services/            # 纯业务逻辑(单据/计价/状态机,可单测)
│   ├── models/              # ORM 模型 users/items/orders/deposits/payments
│   ├── payments/            # 微信支付网关(预付/回调/退款/签名)
│   ├── search/              # go-scout 检索(索引/删除/半径)
│   ├── middleware/          # JWT 双 Token、security-go
│   ├── static/              # 静态资源(吉祥物 mascot.svg, GET /static/mascot.svg)
│   └── *_test.go            # 离线单测,免基建
├── deploy/                  # Docker Compose(dev/prod)、Schema、镜像
├── docs/                    # 架构/功能/生命周期 SVG、支付码、配置与规划文档
├── scripts/push-release.sh  # 推送规则:增量 tag + GitHub Release
└── .github/workflows/ci.yml # CI: lint + build + vet + test + 覆盖率门
```

## 使用说明

### 环境要求

- Go 1.24+;Docker + Docker Compose(mysql:8、redis:7、opensearch:2.14)
- 微信支付商户密钥(M3 证书;mock 模式不需要)

### 本地开发

```bash
# 1. 起基础设施(dev compose;若宿主机 3306 被占,设 MYSQL_HOST_PORT=3307)
docker compose -f deploy/docker-compose.yml up -d

# 2. 起服务(dev 自动建表 + 启动重建索引;OpenSearch 不可用也不崩)
cd server && go run .
```

冒烟验证(短信为 mock,验证码 `123456`):

```bash
# 登录得 token
curl -s -X POST http://127.0.0.1:8080/api/v1/auth/sms -H 'Content-Type: application/json' \
  -d '{"phone":"13800138000"}'
curl -s -X POST http://127.0.0.1:8080/api/v1/auth/login -H 'Content-Type: application/json' \
  -d '{"phone":"13800138000","code":"123456"}'
# 发布物品(JWT)
curl -s -X POST http://127.0.0.1:8080/api/v1/items -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"单反相机","category_id":1,"daily_price":30,"deposit":200,"stock":1}'
# 公开读详情(无需 token)
curl -s http://127.0.0.1:8080/api/v1/items/1
```

### 测试与构建

```bash
cd server && go build ./... && go vet ./... && go test ./... -count=1   # 离线全绿
```

### 生产部署

```bash
cp deploy/.env.example deploy/.env          # 强密钥/商户证书路径(必设)
docker compose -f deploy/docker-compose.prod.yml up -d   # 端口覆盖 3307/16379/9200
BEEGO_RUNMODE=prod ./server                 # 或容器内运行
```

> 端口占用与 OpenSearch 安全配置等坑位详见 [docs/design.md](docs/design.md) 排查要点。

### 推送与版本发布(推送规则)

```bash
./scripts/push-release.sh            # 推送分支 → 读最新 tag → 增量 patch 版 tag + GitHub Release
./scripts/push-release.sh --minor    # 或 --major / --version v2.0.0
```

规则:读取仓库最高 `v*` tag(无则从 `v0.0.0` 起步)→ 推送当前分支代码 → 增量 bump 版本 → 打注解 tag 并推送 → `gh release create --generate-notes`(自动收录自上一 tag 以来的提交)。需 git 干净工作区,已存在的版本拒绝重复。

## API 概览

`Base URL: /api/v1`;除标注「公开」外均需 `Authorization: Bearer <AccessToken>`。

### 认证 / 用户

| 方法 | 路径 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| POST | /auth/sms | 公开 | 发送验证码(60s 限频) |
| POST | /auth/login | 公开 | 验证码登录(自动注册) |
| POST | /auth/refresh | 公开 | 双 Token 轮换 |
| GET / PUT | /user/profile | JWT | 查看 / 更新资料 |

### 类目 / 物品

| 方法 | 路径 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| GET | /categories | 公开 | 类目列表 |
| GET | /items | 公开 | 物品分页列表(city/category) |
| GET | /items/search | 公开 | 关键词搜索 + 半径检索 |
| GET | /items/:id | 公开 | 物品详情 |
| POST | /items | JWT | 发布物品 |
| PUT | /items/:id | JWT(owner) | 修改物品 |
| POST | /items/:id/offshelf | JWT(owner) | 下架(同步删索引) |

### 订单(全部 JWT)

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | /orders | 创建订单(status=0) |
| GET | /orders | 我的订单 |
| GET | /orders/:id | 订单详情(renter/owner) |
| POST | /orders/:id/pickup | 取货(1→2,冻结押金) |
| POST | /orders/:id/return_request | 申请归还(2→3) |
| POST | /orders/:id/return_confirm | 确认归还(3→4,解冻结算) |
| POST | /orders/:id/breach | 违约(3→6,扣押金) |
| POST | /orders/:id/cancel | 取消(0→5,退款) |

### 支付

| 方法 | 路径 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| POST | /pay/unifiedorder | JWT | 微信预付(JSAPI) |
| POST | /pay/refund | JWT(owner/renter) | 退款 |
| POST | /pay/notify | 回调验签 | 支付/退款结果回调 |

## 相关文档

- [docs/plan.md](docs/plan.md) — 项目规划与路线图
- [docs/task.md](docs/task.md) — 任务清单与状态
- [docs/design.md](docs/design.md) — 设计决策与排查要点
- [docs/api.md](docs/api.md) — API 接口文档(接口/请求/响应/错误码)
- [docs/config-runbook.md](docs/config-runbook.md) — 配置键与部署手册
- [docs/frontend-plan.md](docs/frontend-plan.md) — 前端页面规划(仅规划)

## 支持与打赏

项目由个人维护。如果它对你有帮助,欢迎支持 — 你的每一份心意都是持续维护的动力。💛

| 打赏方式 | 二维码 |
| --- | --- |
| 微信赞赏码 | ![微信赞赏码](docs/support/weixin_qr.png) |
| 支付宝收款码 | ![支付宝收款码](docs/support/alipay_qr.png) |

> 左侧为**微信赞赏码**,右侧为**支付宝收款码**,分别对应微信与支付宝应用内扫码。

也支持虚拟币打赏(下方二维码与收款地址一一对应):

| # | 网络 | 收款地址 |
| --- | --- | --- |
| 1 | BNB Smart Chain (BEP20) | ![BEP20](docs/coin/small/1.jpg) `0x355d429f97511897ccb4e271ec888205f9ab6629` |
| 2 | Tron (TRC20) | ![TRC20](docs/coin/small/2.jpg) `TEdDHWLajt1XvqtPDWmQctdrJaC3pzZZzz` |
| 3 | Ethereum (ERC20) | ![ERC20](docs/coin/small/3.jpg) `0x355d429f97511897ccb4e271ec888205f9ab6629` |
| 4 | Aptos | ![Aptos](docs/coin/small/4.jpg) `0x836e3780edfc3f7b2372b39e2a1a3a5d7adfaccd96c726f21cfde1b50dd68030` |
| 5 | Plasma | ![Plasma](docs/coin/small/5.jpg) `0x355d429f97511897ccb4e271ec888205f9ab6629` |
| 6 | Polygon POS | ![Polygon](docs/coin/small/6.jpg) `0x355d429f97511897ccb4e271ec888205f9ab6629` |
| 7 | Solana | ![Solana](docs/coin/small/7.jpg) `2hfhboHdmdrYsY25XfQSsEWxq5ip4EQsR7f4AzSRMUyr` |
| 8 | The Open Network (TON) | ![TON](docs/coin/small/8.jpg) `UQB9kFQohzmXUir9QSSZq01iwl9aQZIDdBpNmDklljRtCoGK` |
| 9 | Arbitrum One | ![Arbitrum](docs/coin/small/9.jpg) `0x355d429f97511897ccb4e271ec888205f9ab6629` |
| 10 | AVAX C-Chain | ![AVAX](docs/coin/small/10.jpg) `0x355d429f97511897ccb4e271ec888205f9ab6629` |

## License

[MIT](./LICENSE) © 2026 erik.xyz

---

*Generated with https://erik.xyz*