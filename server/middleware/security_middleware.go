// security-go 安全中间件：对每个请求做攻击检测，命中高危/严重直接拦截并计入封禁
package middleware

import (
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/beego/beego/v2/server/web/context"
	sec "github.com/erikwang2013/item-rental/server/security"
	"github.com/erikwang2013/security-go"
)

const (
	// blockLevel 拦截阈值：达到该严重程度及以上的攻击直接拦截
	blockLevel = security.SeverityHigh
)

// SecurityFilter 是 beego 前置过滤器，对请求做全量攻击检测。
// 语义分离：
//  1. 命中高危/严重攻击 → 一律 403 拦截（WAF 本职）
//  2. 拦截的同时，对来源 IP 计数，达到阈值触发自动封禁
//  3. 已封禁的 IP → 直接拒绝
func SecurityFilter(ctx *context.Context) {
	if sec.Engine == nil {
		return
	}

	clientIP := ClientIP(ctx.Request)

	// 1. 已封禁的 IP 直接拒绝
	if sec.IPBlacklist.Detect(clientIP).Detected {
		reject(ctx, "IP 已被封禁")
		return
	}

	// 2. 全量攻击检测。
	// 从待扫描的请求副本中剥离 Authorization 头：客户端必须通过该头传递 JWT，
	// 若原样送入引擎，SensitiveDataLeak(data_leak) 会把合法 JWT 误判为"敏感数据泄露"
	// (SeverityCritical) 导致所有鉴权端点一律 403。攻击检测不需要 Authorization 头。
	scanReq := ctx.Request.Clone(ctx.Request.Context())
	scanReq.Header.Del("Authorization")
	results := sec.Engine.DetectRequest(scanReq)
	attacked := false
	for _, r := range results {
		if r.Detected && r.Severity >= blockLevel {
			attacked = true
			break
		}
	}
	if !attacked {
		return
	}

	// 3. 计数并判断是否触发封禁（RecordAttack 的 bool 表示"本次是否达到阈值"）
	blocked, err := sec.IPBlacklist.RecordAttack(clientIP)
	if err != nil {
		log.Printf("[security] RecordAttack(%s) failed: %v", clientIP, err)
	}
	// 无论是否触发封禁，高危攻击都直接拦截
	if blocked {
		reject(ctx, "检测到恶意请求，IP 已被封禁")
		return
	}
	reject(ctx, "检测到恶意请求，已拦截")
}

// reject 以 403 拒绝请求
func reject(ctx *context.Context, msg string) {
	ctx.Output.SetStatus(http.StatusForbidden)
	_ = ctx.Output.JSON(map[string]any{"code": 403, "msg": msg}, false, false)
	ctx.Abort(403, "Forbidden")
}

// trustedProxyCIDRs 可信反代 IP 段（如 Nginx/网关）。
// 仅当连接来自这些来源时才信任 X-Forwarded-For / X-Real-IP，否则回退到 RemoteAddr，
// 防止客户端伪造请求头绕过 IP 封禁。生产环境请按实际部署配置。
var trustedProxyCIDRs = parseCIDRs(envCSV("ITEM_RENTAL_TRUSTED_PROXY"))

// ClientIP 提取真实客户端 IP。
// 安全原则：只有来自可信反代的连接才信任 X-Forwarded-For / X-Real-IP；
// 其他情况一律以 TCP 对端地址 RemoteAddr 为准。
func ClientIP(r *http.Request) string {
	remote := remoteIP(r.RemoteAddr)
	if isTrustedProxy(remote) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			return strings.TrimSpace(parts[0])
		}
		if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
			return strings.TrimSpace(xrip)
		}
	}
	return remote
}

// remoteIP 从 "host:port" 提取 IP。
func remoteIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return host
	}
	return remoteAddr
}

// isTrustedProxy 判断来源 IP 是否属于可信反代。
// 未配置可信反代时（len==0），为安全起见一律不信任转发头。
func isTrustedProxy(ip string) bool {
	if len(trustedProxyCIDRs) == 0 {
		return false
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, cidr := range trustedProxyCIDRs {
		if cidr.Contains(parsed) {
			return true
		}
	}
	return false
}

// envCSV 读取环境变量并拆分为逗号分隔的非空字符串列表。
func envCSV(key string) []string {
	raw := os.Getenv(key)
	if raw == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// parseCIDRs 将 CIDR 字符串列表解析为 *net.IPNet 列表，跳过非法项。
func parseCIDRs(cidrs []string) []*net.IPNet {
	var out []*net.IPNet
	for _, c := range cidrs {
		_, ipNet, err := net.ParseCIDR(c)
		if err != nil {
			// 也支持纯 IP
			if ip := net.ParseIP(c); ip != nil {
				bits := 32
				if ip.To4() == nil {
					bits = 128
				}
				_, ipNet, err = net.ParseCIDR(c + "/" + strconv.Itoa(bits))
				if err != nil {
					continue
				}
			} else {
				continue
			}
		}
		out = append(out, ipNet)
	}
	return out
}
