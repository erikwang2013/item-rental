// security-go 集成：攻击检测引擎初始化与全局访问
package security

import (
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/server/web"
	"github.com/erikwang2013/security-go"
	"github.com/erikwang2013/security-go/all"
	"github.com/erikwang2013/security-go/httpval"
	"github.com/erikwang2013/security-go/storage"
)

var (
	// Engine 全局攻击检测引擎
	Engine *security.Engine
	// IPBlacklist 自动封禁器
	IPBlacklist *httpval.IPBlacklist

	initOnce sync.Once
)

// InitEngine 初始化攻击检测引擎，注册全部检测器与 HTTP 校验器。
// 应在应用启动时调用（routers init 中）。sync.Once 保证只初始化一次，
// 避免并发/重复调用造成数据竞态。
func InitEngine() {
	initOnce.Do(func() {
		e := security.NewEngine()

		// 注册 27 个零配置检测器：注入/协议/数据/文件
		all.RegisterAll(e)

		// IP 黑名单自动封禁（默认 5 次/60s -> 封禁 15 分钟，可配置）
		threshold := int(web.AppConfig.DefaultInt("ipban_threshold", 5))
		window := time.Duration(web.AppConfig.DefaultInt("ipban_window", 60)) * time.Second
		ban := time.Duration(web.AppConfig.DefaultInt("ipban_duration", 900)) * time.Second

		// 使用内存后端；生产环境可替换为 Redis 后端（见 storage 包）
		mem := storage.NewMemory()
		bl := httpval.NewIPBlacklist(mem)
		bl.Threshold = threshold
		bl.Window = window
		bl.BanDuration = ban
		e.Register(bl)

		// 请求体大小上限（默认 10MB）
		bodyLimit := web.AppConfig.DefaultInt("body_size_limit", 10*1024*1024)
		e.Register(httpval.NewBodySize(int64(bodyLimit)))

		// Content-Type 白名单
		cts := web.AppConfig.DefaultString("content_types", "application/json,application/x-www-form-urlencoded,multipart/form-data")
		e.Register(httpval.NewContentType(splitCSV(cts)))

		// HTTP 方法校验
		e.Register(&httpval.Method{})

		// 全部初始化完成后发布（先构建局部变量，最后一次性赋值给全局）
		Engine = e
		IPBlacklist = bl
	})
}

// splitCSV 拆分逗号分隔的配置字符串并去除空白。
func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
