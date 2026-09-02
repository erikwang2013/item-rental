// 微信支付网关：统一下单（Native 扫码 + JSAPI）与结果处理
package payments

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

// 统一下单接口地址（V2 XML 协议）
const unifiedOrderURL = "https://api.mch.weixin.qq.com/pay/unifiedorder"

// 退款接口地址（V2 XML 协议，需商户证书双向 TLS）
const refundURL = "https://api.mch.weixin.qq.com/secapi/pay/refund"

// 支付渠道
const (
	ChannelNative = "native" // 扫码支付
	ChannelJSAPI  = "jsapi"  // 公众号/小程序支付
)

// 统一下单结果
type PrepayResult struct {
	// PrepayID 微信统一下单返回的预支付交易会话标识
	PrepayID string `json:"prepay_id"`
	// CodeURL Native 扫码支付链接
	CodeURL string `json:"code_url,omitempty"`
	// PayParams JSAPI 拉起支付所需的参数（含二次签名）
	PayParams map[string]string `json:"pay_params,omitempty"`
	// Mock 是否 mock 模式返回
	Mock bool `json:"mock"`
}

// gateway 微信支付网关实现
type gateway struct {
	cfg Config
	svc OrderService
}

// Gateway 支付网关接口
type Gateway interface {
	// CreatePrepay 创建微信统一下单
	CreatePrepay(req UnifiedOrderReq) (*PrepayResult, error)
	// HandleNotify 处理微信支付回调，返回成功/失败
	HandleNotify(rawXML []byte) error
	// Refund 发起退款。mock 模式直接成功；真实模式调用 secapi/pay/refund（需商户证书）。
	Refund(req RefundReq) (*RefundResult, error)
}

// RefundReq 微信退款请求
type RefundReq struct {
	OutTradeNo string  // 原支付商户订单号（payments.out_trade_no）
	TotalFee   float64 // 原订单支付金额（元）
	RefundFee  float64 // 本次退款金额（元）
	RefundNo   string  // 商户退款单号（out_refund_no，需全局唯一）
}

// RefundResult 退款结果
type RefundResult struct {
	// RefundID 微信退款单号
	RefundID string `json:"refund_id"`
	// Mock 是否 mock 模式返回
	Mock bool `json:"mock"`
}

// UnifiedOrderReq 统一下单请求
type UnifiedOrderReq struct {
	OutTradeNo string  // 商户订单号（与 payments.out_trade_no 一致）
	Body       string  // 商品描述
	Amount     float64 // 应付金额（元）
	Channel    string  // native / jsapi
	OpenID     string  // JSAPI 支付必传
	ClientIP   string  // 用户终端 IP
}

// NewGateway 构建支付网关。
// 依赖 config 与 order service（用于支付成功后标记订单已支付）。
func NewGateway(cfg Config, svc OrderService) Gateway {
	return &gateway{cfg: cfg, svc: svc}
}

// DefaultGateway 使用默认配置与默认订单服务构建网关（便于测试/简单调用）。
func DefaultGateway() Gateway {
	return NewGateway(LoadConfig(), NewOrderService())
}

// CreatePrepay 统一下单。
// 根据 channel 区分 Native / JSAPI，组装 XML 请求并签名。
func (g *gateway) CreatePrepay(req UnifiedOrderReq) (*PrepayResult, error) {
	if req.OutTradeNo == "" || req.Amount <= 0 {
		return nil, errors.New("下单参数错误")
	}
	if req.Channel == "" {
		req.Channel = ChannelNative
	}
	if req.Channel != ChannelNative && req.Channel != ChannelJSAPI {
		return nil, errors.New("不支持的支付渠道")
	}
	if req.Channel == ChannelJSAPI && req.OpenID == "" {
		return nil, errors.New("JSAPI 支付需要 openid")
	}
	if req.ClientIP == "" {
		req.ClientIP = "127.0.0.1"
	}

	// mock 模式：不真实调用微信，直接返回模拟 prepay_id
	if g.cfg.Mock {
		return g.mockPrepay(req), nil
	}

	// 组装统一下单参数（金额转分，整型）
	params := map[string]string{
		"appid":            g.cfg.AppID,
		"mch_id":           g.cfg.MchID,
		"nonce_str":        nonceStr(),
		"body":             req.Body,
		"out_trade_no":     req.OutTradeNo,
		"total_fee":        fmt.Sprintf("%d", fen(req.Amount)),
		"spbill_create_ip": req.ClientIP,
		"notify_url":       g.cfg.NotifyURL,
		"trade_type":       req.Channel,
	}
	if req.Channel == ChannelJSAPI {
		params["openid"] = req.OpenID
	}
	// 签名（sign 字段本身不参与计算，sign_type 参与）
	params["sign_type"] = g.cfg.SignType
	params["sign"] = signParams(params, g.cfg.MchKey, g.cfg.SignType)

	xmlBody, err := mapToXML(params)
	if err != nil {
		return nil, err
	}

	resp, err := g.postXML(unifiedOrderURL, xmlBody)
	if err != nil {
		return nil, err
	}
	if !verifySign(resp, g.cfg.MchKey, g.cfg.SignType) {
		return nil, errors.New("微信下单应答签名校验失败")
	}
	if resp["return_code"] != "SUCCESS" {
		return nil, fmt.Errorf("下单失败: %s", resp["return_msg"])
	}

	res := &PrepayResult{PrepayID: resp["prepay_id"]}
	switch req.Channel {
	case ChannelNative:
		res.CodeURL = resp["code_url"]
	case ChannelJSAPI:
		res.PayParams = jsapiPayParams(g.cfg, resp["prepay_id"])
	}
	return res, nil
}

// mockPrepay mock 模式统一下单：返回固定 prepay_id。
func (g *gateway) mockPrepay(req UnifiedOrderReq) *PrepayResult {
	prepayID := fmt.Sprintf("mock_prepay_%s", req.OutTradeNo)
	res := &PrepayResult{PrepayID: prepayID, Mock: true}
	switch req.Channel {
	case ChannelNative:
		res.CodeURL = "weixin://wxpay/bizpayurl?pr=mock"
	case ChannelJSAPI:
		res.PayParams = jsapiPayParams(g.cfg, prepayID)
	}
	return res
}

// Refund 发起退款。
// mock 模式：不真实调用微信，直接返回模拟 refund_id。
// 真实模式：调用 secapi/pay/refund（需商户证书双向 TLS），
// 证书未配置时返回明确错误，不 panic。
func (g *gateway) Refund(req RefundReq) (*RefundResult, error) {
	if req.OutTradeNo == "" || req.RefundNo == "" || req.TotalFee <= 0 || req.RefundFee <= 0 {
		return nil, errors.New("退款参数错误")
	}
	if req.RefundFee > req.TotalFee {
		return nil, errors.New("退款金额不能大于订单金额")
	}

	// mock 模式：直接成功，便于本地/联调环境跑通退款链路
	if g.cfg.Mock {
		return g.mockRefund(req), nil
	}

	// 退款接口需要商户证书（双向 TLS），未配置给出明确错误而非 panic
	if g.cfg.CertFile == "" || g.cfg.CertKey == "" {
		return nil, errors.New("微信退款需配置商户证书: wechat_cert_file / wechat_cert_key (或 WECHAT_CERT_FILE / WECHAT_CERT_KEY)")
	}

	params := map[string]string{
		"appid":         g.cfg.AppID,
		"mch_id":        g.cfg.MchID,
		"nonce_str":     nonceStr(),
		"out_trade_no":  req.OutTradeNo,
		"total_fee":     fmt.Sprintf("%d", fen(req.TotalFee)),
		"refund_fee":    fmt.Sprintf("%d", fen(req.RefundFee)),
		"out_refund_no": req.RefundNo,
		"sign_type":     g.cfg.SignType,
	}
	params["sign"] = signParams(params, g.cfg.MchKey, g.cfg.SignType)

	xmlBody, err := mapToXML(params)
	if err != nil {
		return nil, err
	}

	resp, err := g.postXMLWithCert(refundURL, xmlBody)
	if err != nil {
		return nil, err
	}
	if !verifySign(resp, g.cfg.MchKey, g.cfg.SignType) {
		return nil, errors.New("微信退款应答签名校验失败")
	}
	if resp["return_code"] != "SUCCESS" || resp["result_code"] != "SUCCESS" {
		return nil, fmt.Errorf("退款失败: %s", resp["return_msg"])
	}
	return &RefundResult{RefundID: resp["refund_id"]}, nil
}

// mockRefund mock 模式退款：返回固定 refund_id。
func (g *gateway) mockRefund(req RefundReq) *RefundResult {
	return &RefundResult{RefundID: fmt.Sprintf("mock_refund_%s", req.RefundNo), Mock: true}
}

// jsapiPayParams 生成 JSAPI 拉起支付所需的二次签名参数。
// paySign 使用统一下单的签名算法对拉起参数签名（字段名与微信 JSAPI 约定一致）。
func jsapiPayParams(cfg Config, prepayID string) map[string]string {
	now := fmt.Sprintf("%d", time.Now().Unix())
	params := map[string]string{
		"appId":     cfg.AppID,
		"timeStamp": now,
		"nonceStr":  nonceStr(),
		"package":   "prepay_id=" + prepayID,
		"signType":  cfg.SignType,
	}
	params["paySign"] = signParams(params, cfg.MchKey, cfg.SignType)
	return params
}

// postXML 发送 XML POST 请求并解析应答。
func (g *gateway) postXML(url string, body []byte) (map[string]string, error) {
	client := &http.Client{
		Timeout: time.Duration(g.cfg.Timeout) * time.Second,
		// 微信支付使用合法 CA 证书，无需跳过校验
		Transport: &http.Transport{TLSClientConfig: &tls.Config{}},
	}
	resp, err := client.Post(url, "text/xml", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return xmlToMap(data)
}

// postXMLWithCert 发送带商户客户端证书的 XML POST（退款等需双向 TLS 的接口）。
func (g *gateway) postXMLWithCert(url string, body []byte) (map[string]string, error) {
	cert, err := tls.LoadX509KeyPair(g.cfg.CertFile, g.cfg.CertKey)
	if err != nil {
		return nil, fmt.Errorf("加载商户证书失败: %w", err)
	}
	client := &http.Client{
		Timeout: time.Duration(g.cfg.Timeout) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{Certificates: []tls.Certificate{cert}},
		},
	}
	resp, err := client.Post(url, "text/xml", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return xmlToMap(data)
}

// xmlToMap 解析微信 XML 应答为 map。
func xmlToMap(data []byte) (map[string]string, error) {
	var m struct {
		XMLName xml.Name  `xml:"xml"`
		Items   []xmlNode `xml:",any"`
	}
	if err := xml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(m.Items))
	for _, it := range m.Items {
		out[it.XMLName.Local] = it.Value
	}
	return out, nil
}

// mapToXML 将参数转为微信要求的 XML（CDATA 包裹）。
func mapToXML(params map[string]string) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString("<xml>")
	for k, v := range params {
		if v == "" {
			continue
		}
		b.WriteString("<" + k + "><![CDATA[" + v + "]]></" + k + ">")
	}
	b.WriteString("</xml>")
	return b.Bytes(), nil
}

// xmlNode XML 通用节点
type xmlNode struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

// fen 元转分（四舍五入到分）。
func fen(yuan float64) int64 {
	return int64(yuan*100 + 0.5)
}

// nonceStr 生成随机字符串。
func nonceStr() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 32)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
