#!/usr/bin/env bash
# 阶段E 端到端冒烟(可执行真源;叙述性记录见 docs/e2e-smoke-d.md)。
# 链1 = 阶段D 主链路(登录/owner 视图/头像/下单/签名 notify/流转/消息/real_name)
# 链2 = cancel+breach(退款守卫 409/取消扣分/违约押金入物主账/违约扣分/站内信)
# 依赖: curl + python3(GNU date)。信封 {code,msg,data}:业务 Fail(code) 为 HTTP 200+body code;
#       中间件 JWTAuth 失败为真 HTTP 401。
# env:
#   BASE_URL   默认 http://127.0.0.1:8080
#   SERVER_PID 外部起服时传入,脚本退出时 kill(可空)
#   REBUILD_DB=1 时: DROP DATABASE rental 重建(001 全量),须 mysql root:root123456 可达
# 用法: bash scripts/e2e-smoke.sh
set -euo pipefail

B="${BASE_URL:-http://127.0.0.1:8080}/api/v1"
PASS=0; FAILS=0
say()  { printf '%s\n' "$*"; }
ok()   { PASS=$((PASS+1)); printf 'PASS %-4s %s\n' "$LINENO" "$1"; }
die()  { FAILS=$((FAILS+1)); printf 'FAIL %-4s %s\n' "$LINENO" "$1"; exit 1; }

# json 取值 / 信封断言: jget <json> <dot-path(dict 键或 list 下标)>; code_of <json>
jget() { python3 -c "import json,sys;d=json.loads(sys.argv[1])
for k in sys.argv[2].split('.'):
 if isinstance(d,list): d=d[int(k)] if k.isdigit() else [x[k] for x in d]
 else: d=d[k]
print(d if not isinstance(d,list) else json.dumps(d))" "$1" "$2"; }
code_of() { jget "$1" code; }

# http <方法> <路径> [额外curl参数...] → 全局 $RES(body) $HTTP(code)。单次请求(勿双发,POST 有副作用)。
http() {
  local m=$1 p=$2 bf; shift 2
  bf=$(mktemp)
  HTTP=$(curl -s -o "$bf" -w '%{http_code}' -X "$m" "$B$p" "${@}")
  RES=$(cat "$bf"); rm -f "$bf"
}

# 登录:业务 400(冷启动窗口)重试 5×4s;输出 access_token
login() {
  local phone=$1 i tok
  for i in 1 2 3 4 5; do
    RES=$(curl -s -X POST "$B/auth/login" -H 'Content-Type: application/json' \
      -d "{\"phone\":\"$phone\",\"code\":\"123456\"}")
    if [ "$(code_of "$RES")" = 0 ]; then tok=$(jget "$RES" data.access_token); echo "$tok"; return; fi
    [ "$i" = 5 ] && { printf '登录失败(%s): %s\n' "$phone" "$RES" >&2; exit 1; }
    sleep 4
  done
}
prof() { # prof <token> → uid credit deposit(unread 消息数)
  http GET /user/profile -H "Authorization: Bearer $1"
  UID=$(jget "$RES" data.id); CREDIT=$(jget "$RES" data.credit_score); DEPOSIT=$(jget "$RES" data.deposit_bal)
}
unread() { http GET /messages -H "Authorization: Bearer $1"; jget "$RES" data.unread; }

TOK_JSON='{"return_code":"SUCCESS","result_code":"SUCCESS","out_trade_no":"%s","transaction_id":"TX%s","total_fee":"%s","nonce_str":"smoke1"}'
notify_xml() { # notify_xml <out_trade_no> <fen> → 打 notify 并断言 SUCCESS
  local otn=$1 fen=$2 sign xml
  sign=$(OTN=$otn FEN=$fen python3 - <<'PY'
import os,hmac,hashlib
p={"return_code":"SUCCESS","result_code":"SUCCESS","out_trade_no":os.environ["OTN"],
   "transaction_id":"TX"+os.environ["OTN"],"total_fee":os.environ["FEN"],"nonce_str":"smoke1"}
s="&".join(f"{k}={p[k]}" for k in sorted(p))+"&key="
print(hmac.new(b"",s.encode(),hashlib.sha256).hexdigest().upper())
PY
)
  xml=$(printf '<xml><return_code><![CDATA[SUCCESS]]></return_code><result_code><![CDATA[SUCCESS]]></result_code><out_trade_no><![CDATA[%s]]></out_trade_no><transaction_id><![CDATA[TX%s]]></transaction_id><total_fee><![CDATA[%s]]></total_fee><nonce_str><![CDATA[smoke1]]></nonce_str><sign><![CDATA[%s]]></sign></xml>' "$otn" "$otn" "$fen" "$sign")
  RES=$(curl -s -X POST "$B/pay/notify" -H 'Content-Type: text/xml' -d "$xml")
  case "$RES" in *SUCCESS*) ;; *) die "notify 未确认: $RES";; esac
}

# 建一张 1x1 PNG(头像/物品图上传用)
printf '%s' 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==' | base64 -d > /tmp/e2e-smoke.png

say '== 环境 =='
if [ "${REBUILD_DB:-0}" = 1 ]; then
  mysql -uroot -proot123456 -e "DROP DATABASE IF EXISTS rental; CREATE DATABASE rental DEFAULT CHARACTER SET utf8mb4;"
  mysql -uroot -proot123456 rental < "$(dirname "$0")/../deploy/initdb/001_schema.sql"
  say 'DB 已重建'
fi
[ -n "${SERVER_PID:-}" ] && trap 'kill $SERVER_PID 2>/dev/null || true' EXIT

say '== 链1 =='
# 1 双号登录 + profile(带 token 200 / 无 token 真 401)
A=$(login 13800138001); B2=$(login 13800138002)
http GET /user/profile -H "Authorization: Bearer $A"
[ "$(code_of "$RES")" = 0 ] && [ -n "$(jget "$RES" data.nickname)" ] || die "profile 带 token 失败"
http GET /user/profile; [ "$HTTP" = 401 ] || die "profile 无 token 应真 401,实际 $HTTP"
ok 'profile 鉴权 200/401'
http GET /user/profile -H "Authorization: Bearer $A"; UID_A=$(jget "$RES" data.id)

# 2 发布带图(真上传 → JSON 数组)
# mkbody <标题> <urls空格串> → images 为 JSON 数组文本字符串(server 契约: string 字段内 JSON)
mkbody() { python3 -c "import json,sys
imgs=sys.argv[2].split() if len(sys.argv)>2 and sys.argv[2] else []
b={'title':sys.argv[1],'category_id':1,'daily_price':30,'deposit':200,'stock':1,'city':'上海'}
if imgs: b['images']=json.dumps(imgs)
print(json.dumps(b,ensure_ascii=False))" "$1" "${2:-}"; }
http POST /items/upload -H "Authorization: Bearer $A" -F "files=@/tmp/e2e-smoke.png"
[ "$(code_of "$RES")" = 0 ] || die "items/upload 失败: $RES"
IMG1=$(jget "$RES" data.urls.0)
http POST /items -H "Authorization: Bearer $A" -H 'Content-Type: application/json' \
  -d "$(mkbody "冒烟-相机-$RANDOM" "$IMG1")"
[ "$(code_of "$RES")" = 0 ] || die "带图发布失败(契约): $RES"
ID_UI=$(jget "$RES" data.id)
python3 -c "import sys;v=sys.argv[1];assert v.isdigit() and len(v)>=17, v" "$ID_UI" || die "id 非 snowflake 字符串: $ID_UI"
ok 'snowflake id ≥17 位字符串'
ok '发布(images JSON 数组文本)'

# 3 owner 视图 4 语义(list 断言走 python,JSON 键与空格不敏感)
list_has() { python3 -c "import json,sys;d=json.loads(sys.argv[1]);print(any(str(i['id'])==sys.argv[2] for i in d['data']['items']))" "$RES" "$1"; }
list_status() { python3 -c "import json,sys;d=json.loads(sys.argv[1]);
print(next((i['status'] for i in d['data']['items'] if str(i['id'])==sys.argv[2]),'missing'))" "$RES" "$1"; }
http GET "/items?owner_id=$UID_A&page_size=100" -H "Authorization: Bearer $A"
[ "$(list_has "$ID_UI")" = True ] || die "owner 视图不含新发布"
http POST "/items/$ID_UI/offshelf" -H "Authorization: Bearer $A" >/dev/null || true
http GET "/items?owner_id=$UID_A&page_size=100" -H "Authorization: Bearer $A"
[ "$(list_status "$ID_UI")" = 0 ] || die "owner 视图下架后应见 status=0"
http GET "/items?owner_id=$UID_A" -H "Authorization: Bearer $B2"
[ "$(code_of "$RES")" = 403 ] || die "他人 owner 视图应 403,实际 $(code_of "$RES")"
http GET "/items?owner_id=$UID_A"
[ "$(code_of "$RES")" = 401 ] || die "无 token owner 视图应 401,实际 $(code_of "$RES")"
http GET "/items?page_size=5"
[ "$(list_has "$ID_UI")" = False ] || die "公开列表含已下架品"
ok 'owner 视图 4 语义'

# 4 头像上传(png 200 + gif 400)
http POST /user/avatar -H "Authorization: Bearer $A" -F "file=@/tmp/e2e-smoke.png"
[ "$(code_of "$RES")" = 0 ] || die "头像上传失败: $RES"
AVURL=$(jget "$RES" data.avatar)
AVPATH="${AVURL#*://}"     # 剥 scheme
AVPATH="/${AVPATH#*/}"     # 剥 host → /static/... 路径
HTTPC=$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:8080$AVPATH")
[ "$HTTPC" = 200 ] || die "头像静态回读 $HTTPC"
printf 'x' > /tmp/e2e-evil.gif
http POST /user/avatar -H "Authorization: Bearer $A" -F "file=@/tmp/e2e-evil.gif"
[ "$(code_of "$RES")" != 0 ] || die "gif 应 400"
ok '头像 png/gif'

# 5 下单→notify→流转
http POST /items -H "Authorization: Bearer $A" -H 'Content-Type: application/json' \
  -d '{"title":"冒烟-可租-'$RANDOM'","category_id":1,"daily_price":30,"deposit":200,"stock":1,"city":"上海"}'
ID=$(jget "$RES" data.id)
S=$(date -d '+3 days' +%F); E=$(date -d '+6 days' +%F)
http POST /orders -H "Authorization: Bearer $B2" -H 'Content-Type: application/json' \
  -d "{\"item_id\":\"$ID\",\"start_date\":\"$S\",\"end_date\":\"$E\"}"
[ "$(code_of "$RES")" = 0 ] || die "下单失败: $RES"
OID=$(jget "$RES" data.id); ONO=$(jget "$RES" data.order_no)
http POST /pay/unifiedorder -H "Authorization: Bearer $B2" -H 'Content-Type: application/json' \
  -d "{\"order_no\":\"$ONO\",\"channel\":\"native\"}"
[ "$(code_of "$RES")" = 0 ] || die "unifiedorder 失败: $RES"
OTN=$(jget "$RES" data.out_trade_no); AMT=$(jget "$RES" data.amount)
FEN=$(python3 -c "print(int(float('$AMT')*100+0.5))")
notify_xml "$OTN" "$FEN"
http GET "/orders/$OID" -H "Authorization: Bearer $B2"
[ "$(jget "$RES" data.status)" = 1 ] || die "notify 后应 status=1"
http POST "/orders/$OID/pickup" -H "Authorization: Bearer $B2" >/dev/null || die pickup
http POST "/orders/$OID/return_request" -H "Authorization: Bearer $B2" >/dev/null || die return_request
http POST "/orders/$OID/return_confirm" -H "Authorization: Bearer $A" >/dev/null || die return_confirm
http GET "/orders/$OID" -H "Authorization: Bearer $B2"
[ "$(jget "$RES" data.status)" = 4 ] || die "应 status=4"
ok 'notify→pickup→return 全链'

# 6 消息 + real_name 回读
UN=$(unread "$B2"); [ "$UN" -ge 1 ] || die "租客未收到 payment_success"
http PUT /user/profile -H "Authorization: Bearer $A" -H 'Content-Type: application/json' \
  -d '{"nickname":"房东-冒烟E","real_name":"张三"}'
[ "$(code_of "$RES")" = 0 ] || die "profile PUT(real_name) 失败"
http GET /user/profile -H "Authorization: Bearer $A"
[ "$(jget "$RES" data.real_name)" = "张三" ] || die "real_name 未回读明文"
ok "消息 unread=$UN + real_name 加密回读"

say '== 链2 =='
# 基准:C 租客(新号 credit=100)/A 房东 deposit 基准
C=$(login 13800138003)
http GET /user/profile -H "Authorization: Bearer $A"; D0=$(jget "$RES" data.deposit_bal); U0A=$(unread "$A")
http GET /user/profile -H "Authorization: Bearer $C"; C0=$(jget "$RES" data.credit_score); U0C=$(unread "$C")
[ "$C0" = 100 ] || die "新租客 credit 应 100,实际 $C0"

# 链2a cancel:下单→notify→退款→回 0→取消(扣 10)+ 消息
http POST /items -H "Authorization: Bearer $A" -H 'Content-Type: application/json' \
  -d '{"title":"冒烟-cancel-'$RANDOM'","category_id":1,"daily_price":30,"deposit":200,"stock":1,"city":"上海"}'
ID=$(jget "$RES" data.id)
http POST /orders -H "Authorization: Bearer $C" -H 'Content-Type: application/json' \
  -d "{\"item_id\":\"$ID\",\"start_date\":\"$S\",\"end_date\":\"$E\"}"
OID=$(jget "$RES" data.id); ONO=$(jget "$RES" data.order_no)
http POST /pay/unifiedorder -H "Authorization: Bearer $C" -H 'Content-Type: application/json' \
  -d "{\"order_no\":\"$ONO\",\"channel\":\"native\"}"
OTN=$(jget "$RES" data.out_trade_no); FEN=$(python3 -c "print(int(float('$(jget "$RES" data.amount)')*100+0.5))")
notify_xml "$OTN" "$FEN"
http POST /pay/refund -H "Authorization: Bearer $C" -H 'Content-Type: application/json' -d "{\"order_id\":\"$OID\"}"
[ "$(code_of "$RES")" = 0 ] || die "退款失败: $RES"
http GET "/orders/$OID" -H "Authorization: Bearer $C"
[ "$(jget "$RES" data.status)" = 0 ] || die "退款后应回 status=0"
http POST "/orders/$OID/cancel" -H "Authorization: Bearer $C" -H 'Content-Type: application/json' \
  -d '{"reason":"冒烟取消"}'
[ "$(code_of "$RES")" = 0 ] || die "取消失败: $RES"
http GET "/orders/$OID" -H "Authorization: Bearer $C"
[ "$(jget "$RES" data.status)" = 5 ] || die "应 status=5"
http GET /user/profile -H "Authorization: Bearer $C"
[ "$(jget "$RES" data.credit_score)" = $((C0-10)) ] || die "已付取消应 -10"
[ "$(unread "$A")" -ge $((U0A+2)) ] || die "房东应收到 refund+cancel 消息"
ok 'cancel 链(退款回 0/取消=5/扣 10/消息)'

# 链2b breach:再下单→notify→pickup→退款 409→return_request→breach(押金入物主+扣 30+消息)
http POST /items -H "Authorization: Bearer $A" -H 'Content-Type: application/json' \
  -d '{"title":"冒烟-breach-'$RANDOM'","category_id":1,"daily_price":30,"deposit":200,"stock":1,"city":"上海"}'
ID=$(jget "$RES" data.id)
http POST /orders -H "Authorization: Bearer $C" -H 'Content-Type: application/json' \
  -d "{\"item_id\":\"$ID\",\"start_date\":\"$S\",\"end_date\":\"$E\"}"
OID=$(jget "$RES" data.id); ONO=$(jget "$RES" data.order_no)
http POST /pay/unifiedorder -H "Authorization: Bearer $C" -H 'Content-Type: application/json' \
  -d "{\"order_no\":\"$ONO\",\"channel\":\"native\"}"
notify_xml "$(jget "$RES" data.out_trade_no)" "$(python3 -c "print(int(float('$(jget "$RES" data.amount)')*100+0.5))")"
http POST "/orders/$OID/pickup" -H "Authorization: Bearer $C" >/dev/null || die pickup
http POST /pay/refund -H "Authorization: Bearer $C" -H 'Content-Type: application/json' -d "{\"order_id\":\"$OID\"}"
[ "$(code_of "$RES")" = 409 ] || die "租赁中退款应 409,实际 $(code_of "$RES")"
http POST "/orders/$OID/return_request" -H "Authorization: Bearer $C" >/dev/null || die return_request
http POST "/orders/$OID/breach" -H "Authorization: Bearer $A" >/dev/null || die breach
http GET "/orders/$OID" -H "Authorization: Bearer $C"
[ "$(jget "$RES" data.status)" = 6 ] || die "应 status=6"
http GET /user/profile -H "Authorization: Bearer $A"
[ "$(python3 -c "print(float('$(jget "$RES" data.deposit_bal)')-float('$D0')>=199.99)")" = True ] || die "违约押金未入物主账"
http GET /user/profile -H "Authorization: Bearer $C"
[ "$(jget "$RES" data.credit_score)" = $((C0-10-30)) ] || die "违约后 credit 应 $((C0-40))"
[ "$(unread "$C")" -ge $((U0C+2)) ] || die "租客应收到 refund(上链)+breach 消息"
ok 'breach 链(409 守卫/押金入账/扣 30/消息)'

say "== 结果: PASS=$PASS FAIL=$FAILS =="
[ "$FAILS" = 0 ]
