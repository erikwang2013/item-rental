// 微信支付签名单元测试：MD5 / HMAC-SHA256 已知向量 + verifySign
package payments

import "testing"

// signParams 的期望值由 Python hashlib/hmac 独立计算（与代码实现无关）。
var signVectors = []struct {
	name     string
	params   map[string]string
	mchKey   string
	signType string
	want     string
}{
	{
		name:     "MD5 忽略 sign 字段与空值",
		params:   map[string]string{"appid": "wx123456", "mch_id": "999888", "total_fee": "100", "sign": "whatever"},
		mchKey:   "mchkey-secret-abc",
		signType: "MD5",
		want:     "8F6964CCDB5CB8C7392348BF25C1C0C3",
	},
	{
		name:     "HMAC-SHA256 已知向量",
		params:   map[string]string{"appid": "wx123456", "mch_id": "999888", "out_trade_no": "T123", "total_fee": "5800", "sign": "y"},
		mchKey:   "mchkey-secret-abc",
		signType: "HMAC-SHA256",
		want:     "F156A570F170CAF5CBD322B8E327D63502404BB236A9229CC3D866970802466A",
	},
	{
		name:     "空值字段不参与拼接",
		params:   map[string]string{"a": "1", "b": "", "sign": "z"},
		mchKey:   "k",
		signType: "MD5",
		want:     "AFFDCC88244C83F871BFE4854BE9C1A5",
	},
}

func TestSignParams(t *testing.T) {
	for _, tt := range signVectors {
		t.Run(tt.name, func(t *testing.T) {
			if got := signParams(tt.params, tt.mchKey, tt.signType); got != tt.want {
				t.Errorf("signParams() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestVerifySign(t *testing.T) {
	// 用已知向量构造合法签名
	params := map[string]string{"appid": "wx123456", "mch_id": "999888", "total_fee": "100"}
	params["sign"] = signVectors[0].want

	if !verifySign(params, signVectors[0].mchKey, "MD5") {
		t.Error("正确签名应通过校验")
	}

	// 篡改金额后签名不再匹配
	params["total_fee"] = "999"
	if verifySign(params, signVectors[0].mchKey, "MD5") {
		t.Error("篡改参数后签名应校验失败")
	}

	// 缺失 sign 字段
	delete(params, "sign")
	if verifySign(params, signVectors[0].mchKey, "MD5") {
		t.Error("缺少 sign 字段应校验失败")
	}

	// 密钥错误
	params["total_fee"] = "100"
	params["sign"] = signVectors[0].want
	if verifySign(params, "wrong-key", "MD5") {
		t.Error("密钥不匹配应校验失败")
	}
}