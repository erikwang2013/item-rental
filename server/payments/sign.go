// 微信支付签名：MD5 / HMAC-SHA256（V2 XML 协议）
package payments

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// signParams 对 map 参数按 key 字典序拼接生成签名串并计算签名。
// WeChat V2 规则：key1=value1&key2=value2...&key=<商户密钥>，忽略空值与 sign 字段。
func signParams(params map[string]string, mchKey, signType string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "sign" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		v := params[k]
		if v == "" {
			continue
		}
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(v)
		b.WriteString("&")
	}
	b.WriteString("key=")
	b.WriteString(mchKey)

	switch signType {
	case "HMAC-SHA256":
		mac := hmac.New(sha256.New, []byte(mchKey))
		mac.Write([]byte(b.String()))
		return strings.ToUpper(hex.EncodeToString(mac.Sum(nil)))
	default: // MD5
		sum := md5.Sum([]byte(b.String()))
		return strings.ToUpper(hex.EncodeToString(sum[:]))
	}
}

// verifySign 校验微信回调/应答签名。
func verifySign(params map[string]string, mchKey, signType string) bool {
	got, ok := params["sign"]
	if !ok || got == "" {
		return false
	}
	expected := signParams(params, mchKey, signType)
	// 固定长度比较，防时序攻击
	return len(got) == len(expected) && hmac.Equal([]byte(got), []byte(expected))
}
