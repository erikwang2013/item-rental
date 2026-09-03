# 阶段D 真实链路冒烟(E2E)— 终态记录 2026-09-04

> 起真后端(curl 级)验证阶段D 能力(owner 列表/头像上传/profile 鉴权)+ 主链路
> (发布→下单→支付 mock→流转→消息)。两轮运行:首轮打出 3 个 A 级缺陷,修复后第二轮全绿。

## 环境准备(一次性)

```bash
# MySQL 3306(本机 root 空密码;skip_name_resolve=OFF,唯一命中 root@localhost)
mysql -uroot -e "CREATE DATABASE IF NOT EXISTS rental DEFAULT CHARACTER SET utf8mb4;"
mysql -uroot -e "ALTER USER 'root'@'localhost' IDENTIFIED BY 'root123456'; FLUSH PRIVILEGES;"  # 对齐 app.conf,冒烟后还原!
mysql -uroot -proot123456 rental < deploy/initdb/001_schema.sql   # 幂等:8表+10品类

# 启动(必须带 JWT secret,默认值 fail-fast;WECHAT_MOCK=1;在 server/ 目录起,静态/上传以 CWD 为基准)
cd server && ITEM_RENTAL_JWT_SECRET=rental-dev-smoke-2026 WECHAT_MOCK=1 go run main.go
curl -s http://127.0.0.1:8080/health   # {"code":0,"msg":"ok"}
# Redis AUTH 噪音日志(mock 降级不阻塞);OpenSearch 缺省 → 跳过 /items/search

# 冒烟结束必须还原(即使失败):
#   mysql -uroot -proot123456 -e "ALTER USER 'root'@'localhost' IDENTIFIED BY ''; FLUSH PRIVILEGES;"
```

## 断言序列与终态结果(第二轮,修复后;双号 A=13800138011 房东/B=13800138012 租客)

信封 `{code,msg,data}`;controller Fail(401/403/409)为 HTTP 200+body code;中间件 JWTAuth 失败为真 HTTP 401。

| # | 断言 | 结果 | 说明 |
|---|------|------|------|
| 1 | 双号登录 | ✅ | code:0 双 token |
| 2a/2b | profile 带 token 200 / 无 token 真 401 | ✅ | S3 鉴权挂载 |
| 3 | 发布物品 | ✅ | |
| 4a | owner 视图:本人可见上架品 | ✅ | fix① 后本人 uid 正常识别 |
| 4b | 下架后 owner 仍见 status=0 | ✅ | |
| 4c | 他人(带 token)→ 403「无权」 | ✅ | 业务文案,非 WAF |
| 4d | 无 token → code 401 | ✅ | controller Fail 信封 |
| 4e | 公开列表不含已下架 | ✅ | |
| 5a | 头像上传 png → URL + 静态回读 200 | ✅ | fix② 后 multipart 放行;落盘 `static/uploads/avatars/<uid>_<ms>.png` |
| 5b | 坏扩展名 → 400 | ✅ | 达业务校验(此前被 WAF 截胡) |
| 6 | 下单 | ✅ | |
| 7 | unifiedorder(order_no) | ✅ | mock prepay |
| 8 | 签名 notify(HMAC-SHA256 空 key)→ status=1 | ✅ | fix③ 后 MarkPaid 正确落单 |
| 9 | pickup=2 → return_request → return_confirm=4 | ✅ | |
| 10 | 消息 unread≥1(payment_success) | ✅ | unread=1 |
| 11 | profile PUT 后回读 | ✅ | |

**附加验证**:历史卡单自愈 — 首轮缺陷③留下的「payments 成功但 orders status=0」订单(orders id=1),重发正确 notify 后 status 0→1 ✅(fix③ 移除幂等早退的效果)。

## 首轮打出的缺陷与修复(均已闭环复验)

1. **S1 owner 视图恒 401**(阶段D 新代码):GET /items 公开,JWTAuth 不执行,controller 内 GetUserID 恒空。修:`item_controller.go` List owner_id>0 时显式 `middleware.JWTAuth`(401 Abort)。复验 4a-4e ✅
2. **WAF 误判一切 multipart 上传**(security-go mail_header:`boundary=`/`Content-Type: multipart/` SeverityCritical;扫描含请求自身 Content-Type;403 计攻击 5次/60s→封 900s)。修:security_middleware.go 扫描副本剥离 multipart Content-Type(镜像 34185ca Authorization 处理)+ 回归 TestMultipartUploadNotBlocked。复验 5a/5b ✅
3. **notify→MarkPaid 传错标识符,订单卡死待支付**(阶段B 遗留,钱路):`notify.go` 传 out_trade_no(RENT 前缀)给按 order_no(ORD 前缀)查单的 MarkPaid → 查无单报错但支付已先落库 → 重试幂等早退 → 订单永久卡 0。修:经 pay.OrderId 读订单取 order_no 再传 MarkPaid;移除两处幂等早退使 MarkPaid 恒执行(自愈历史卡单)。复验 8 ✅ + 旧单自愈 ✅

## 残余观察(未修,记录)

- **登录冷启动间歇 400**:两轮服务启动后 ~1-2 分钟内,登录请求间歇返回业务 400「手机号和验证码不能为空」(BindJSON 读到空 body);窗口过后长期稳定(连续 15+ 通过)。curl 直连/管道/换号/换 UA/去代理均不能稳定复现,未定位根因(疑 beego v2.3.10 copyrequestbody 冷启动竞态)。冒烟脚本对登录做 5 次×4s 重试即可绕过。若线上出现,先查 beego Input.RequestBody 拷贝与前置 filter 的交互。

## 裁切标注

- 取消/退款/违约分支未跑(退款 mock 有单测覆盖;主链已覆盖支付+两角色流转)。
- `/items/search`(无 SCOUT_DRIVER 降级)、refresh 轮换(Redis AUTH 噪音)跳过。
- wxapp/flutter 端 UI 冒烟未做(本期后端契约冒烟;两端静态验证门各自通过)。
- 数据库残留首轮冒烟数据(用户 13800138001/2、物品、orders id1-2、payments);冒烟库可随时 DROP 重建(001_schema 幂等)。
