// 支付网关单元测试：金额换算、XML 编解码、mock 下单、回调验签前置分支
package payments

import (
	"strings"
	"testing"
)

func TestFen(t *testing.T) {
	cases := []struct {
		yuan float64
		want int64
	}{
		{0, 0},
		{1, 100},
		{58.8, 5880},
		{10.005, 1001}, // 四舍五入到分
		{0.01, 1},
	}
	for _, c := range cases {
		if got := fen(c.yuan); got != c.want {
			t.Errorf("fen(%v) = %d, want %d", c.yuan, got, c.want)
		}
	}
}

func TestParseFen(t *testing.T) {
	if parseFen("5800") != 5800 {
		t.Error("parseFen(\"5800\") 应解析为 5800")
	}
	// 注：parseFen("12.5") 经 Sscanf %d 会解析为 12（微信不发送非整数金额，非缺陷）
	for _, bad := range []string{"", "abc", "-"} {
		if parseFen(bad) != 0 {
			t.Errorf("parseFen(%q) 应返回 0", bad)
		}
	}
}

func TestMapXMLRoundTrip(t *testing.T) {
	params := map[string]string{"return_code": "SUCCESS", "total_fee": "5800", "empty": ""}
	raw, err := mapToXML(params)
	if err != nil {
		t.Fatalf("mapToXML: %v", err)
	}
	// 空值字段不应出现在 XML 中
	if strings.Contains(string(raw), "empty") {
		t.Error("mapToXML 应跳过空值字段")
	}
	got, err := xmlToMap(raw)
	if err != nil {
		t.Fatalf("xmlToMap: %v", err)
	}
	if got["return_code"] != "SUCCESS" || got["total_fee"] != "5800" {
		t.Errorf("XML round-trip 丢失字段: %v", got)
	}
	if _, err := xmlToMap([]byte("not xml")); err == nil {
		t.Error("非法 XML 应解析失败")
	}
}

func newTestGateway(mock bool) *gateway {
	return &gateway{
		cfg: Config{Mock: mock, AppID: "wx123456", MchID: "999888", MchKey: "mchkey-secret-abc", SignType: "MD5", NotifyURL: "http://localhost/notify"},
		svc: nil,
	}
}

func TestCreatePrepayValidation(t *testing.T) {
	g := newTestGateway(false)
	cases := []struct {
		name string
		req  UnifiedOrderReq
	}{
		{"空订单号", UnifiedOrderReq{Amount: 10}},
		{"金额非正", UnifiedOrderReq{OutTradeNo: "T1"}},
		{"不支持的渠道", UnifiedOrderReq{OutTradeNo: "T1", Amount: 10, Channel: "h5"}},
		{"JSAPI 缺 openid", UnifiedOrderReq{OutTradeNo: "T1", Amount: 10, Channel: ChannelJSAPI}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := g.CreatePrepay(c.req); err == nil {
				t.Error("非法下单参数应返回错误")
			}
		})
	}
}

func TestCreatePrepayMock(t *testing.T) {
	g := newTestGateway(true)

	// Native 扫码
	native, err := g.CreatePrepay(UnifiedOrderReq{OutTradeNo: "T1", Amount: 58.8, Channel: ChannelNative})
	if err != nil {
		t.Fatalf("mock 下单失败: %v", err)
	}
	if !native.Mock || native.PrepayID != "mock_prepay_T1" || native.CodeURL == "" {
		t.Errorf("native mock 结果异常: %+v", native)
	}

	// JSAPI：透传 openid 生成拉起参数
	js, err := g.CreatePrepay(UnifiedOrderReq{OutTradeNo: "T2", Amount: 1, Channel: ChannelJSAPI, OpenID: "openid-1"})
	if err != nil {
		t.Fatalf("jsapi mock 下单失败: %v", err)
	}
	if js.PayParams["package"] != "prepay_id=mock_prepay_T2" || js.PayParams["signType"] != "MD5" {
		t.Errorf("jsapi pay_params 异常: %+v", js.PayParams)
	}
	// paySign 是微信约定的字段名（非 sign），用剩余字段独立重算校验
	paySign := js.PayParams["paySign"]
	expect := map[string]string{
		"appId":     js.PayParams["appId"],
		"timeStamp": js.PayParams["timeStamp"],
		"nonceStr":  js.PayParams["nonceStr"],
		"package":   js.PayParams["package"],
		"signType":  js.PayParams["signType"],
	}
	if paySign == "" || paySign != signParams(expect, g.cfg.MchKey, g.cfg.SignType) {
		t.Error("jsapi paySign 与独立重算结果不一致")
	}
}

// TestHandleNotifyPreDB 覆盖 ormer() 之前的全部校验分支（验签/字段/金额）。
// 后续 DB 分支依赖全局 ORM，离线单测不触及。
func TestHandleNotifyPreDB(t *testing.T) {
	g := newTestGateway(true)

	if err := g.HandleNotify([]byte("<xml>bad</xml>")); err == nil {
		t.Error("非法 XML 应返回错误")
	}

	// 支付结果非成功
	fail, _ := mapToXML(map[string]string{"return_code": "FAIL", "result_code": "FAIL"})
	if err := g.HandleNotify(fail); err == nil {
		t.Error("非 SUCCESS 回调应返回错误")
	}

	// 签名错误
	badSign, _ := mapToXML(map[string]string{"return_code": "SUCCESS", "result_code": "SUCCESS", "sign": "deadbeef"})
	if err := g.HandleNotify(badSign); err == nil {
		t.Error("签名错误的回调应被拒绝")
	}

	// 签名合法但缺关键字段：total_fee 缺失 → 金额非法（在 ormer 之前）
	params := map[string]string{
		"return_code":   "SUCCESS",
		"result_code":   "SUCCESS",
		"out_trade_no":  "T123",
		"transaction_id": "wx111",
	}
	params["sign"] = signParams(params, g.cfg.MchKey, g.cfg.SignType)
	missing, _ := mapToXML(params)
	if err := g.HandleNotify(missing); err == nil {
		t.Error("缺 total_fee 的回调应返回错误")
	}

	// 签名合法但金额非法字符串
	params["total_fee"] = "abc"
	params["sign"] = signParams(params, g.cfg.MchKey, g.cfg.SignType)
	badFee, _ := mapToXML(params)
	if err := g.HandleNotify(badFee); err == nil {
		t.Error("金额非法的回调应返回错误")
	}
}