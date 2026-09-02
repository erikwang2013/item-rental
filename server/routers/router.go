// 路由注册：挂载 security-go 安全中间件、JWT 鉴权中间件与各业务路由
package routers

import (
	stdctx "context"
	"log"

	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
	"github.com/erikwang2013/item-rental/server/controllers"
	"github.com/erikwang2013/item-rental/server/middleware"
	"github.com/erikwang2013/item-rental/server/models"
	"github.com/erikwang2013/item-rental/server/search"
	"github.com/erikwang2013/item-rental/server/security"
)

// requireAuth 注册"方法感知"的 JWT 过滤器。
// Beego 的 InsertFilter 只按 URL 匹配、与 HTTP 方法无关：若直接给 /api/v1/items/:id
// 挂 JWTAuth，会把同路径下公开的 GET 读接口（物品详情/搜索）一并拦截。
// 这里按方法分流：GET（公开读）放行，其余方法（写操作）需通过 JWT 校验。
func requireAuth(path string) {
	web.InsertFilter(path, web.BeforeRouter, func(ctx *context.Context) {
		if ctx.Input.Method() == "GET" {
			return
		}
		middleware.JWTAuth(ctx)
	})
}

func init() {
	// 初始化攻击检测引擎（注册全部检测器）
	security.InitEngine()
	// 初始化数据库连接与模型
	models.InitORM()
	// 初始化搜索引擎（go-scout + OpenSearch）
	search.Init()
	// 异步全量重建搜索索引；OpenSearch 不可达时只记日志、绝不影响启动
	go func() {
		if err := search.ReindexAll(stdctx.Background()); err != nil {
			log.Printf("[search] 启动时全量重建索引失败: %v", err)
		}
	}()

	// 全局安全过滤器：对每个请求做攻击检测 + IP 自动封禁
	// 置于最前，拦截恶意请求
	web.InsertFilter("/*", web.BeforeRouter, middleware.SecurityFilter)

	// 健康检查（公开）
	web.Get("/health", func(ctx *context.Context) {
		_ = ctx.Output.JSON(map[string]any{"code": 0, "msg": "ok"}, false, false)
	})

	// 认证相关（公开）
	web.Router("/api/v1/auth/sms", &controllers.AuthController{}, "post:SendSms")
	web.Router("/api/v1/auth/login", &controllers.AuthController{}, "post:Login")
	web.Router("/api/v1/auth/refresh", &controllers.AuthController{}, "post:Refresh")

	// 用户信息（需登录）
	web.Router("/api/v1/user/profile", &controllers.UserController{}, "get:Profile")
	web.Router("/api/v1/user/profile", &controllers.UserController{}, "put:UpdateProfile")

	// 品类（公开读取）
	web.Router("/api/v1/categories", &controllers.CategoryController{}, "get:List")

	// 物品（公开读取）
	web.Router("/api/v1/items", &controllers.ItemController{}, "get:List")
	web.Router("/api/v1/items/search", &controllers.ItemController{}, "get:Search")
	web.Router("/api/v1/items/:id", &controllers.ItemController{}, "get:Detail")

	// 物品写操作（需登录：发布/更新/下架）；GET 读公开
	requireAuth("/api/v1/items")
	requireAuth("/api/v1/items/:id")
	requireAuth("/api/v1/items/:id/offshelf")
	web.Router("/api/v1/items", &controllers.ItemController{}, "post:Create")
	web.Router("/api/v1/items/:id", &controllers.ItemController{}, "put:Update")
	web.Router("/api/v1/items/:id/offshelf", &controllers.ItemController{}, "post:OffShelf")

	// ---- 订单 ----
	// 订单创建/列表/详情全部需登录（GET 也非公开：仅本人可见自己的订单）
	web.InsertFilter("/api/v1/orders", web.BeforeRouter, middleware.JWTAuth)
	web.InsertFilter("/api/v1/orders/:id", web.BeforeRouter, middleware.JWTAuth)
	web.Router("/api/v1/orders", &controllers.OrderController{}, "post:Create")
	web.Router("/api/v1/orders", &controllers.OrderController{}, "get:List")
	web.Router("/api/v1/orders/:id", &controllers.OrderController{}, "get:Detail")
	// 订单流转（取货/申请归还/确认归还/违约），全部需登录且仅本单租客/房东可操作
	web.InsertFilter("/api/v1/orders/:id/pickup", web.BeforeRouter, middleware.JWTAuth)
	web.InsertFilter("/api/v1/orders/:id/return_request", web.BeforeRouter, middleware.JWTAuth)
	web.InsertFilter("/api/v1/orders/:id/return_confirm", web.BeforeRouter, middleware.JWTAuth)
	web.InsertFilter("/api/v1/orders/:id/breach", web.BeforeRouter, middleware.JWTAuth)
	web.InsertFilter("/api/v1/orders/:id/cancel", web.BeforeRouter, middleware.JWTAuth)
	web.Router("/api/v1/orders/:id/pickup", &controllers.OrderController{}, "post:Pickup")
	web.Router("/api/v1/orders/:id/return_request", &controllers.OrderController{}, "post:ReturnRequest")
	web.Router("/api/v1/orders/:id/return_confirm", &controllers.OrderController{}, "post:ReturnConfirm")
	web.Router("/api/v1/orders/:id/breach", &controllers.OrderController{}, "post:Breach")
	web.Router("/api/v1/orders/:id/cancel", &controllers.OrderController{}, "post:Cancel")

	// ---- M3 微信支付 ----
	// 创建支付单（需登录）
	web.InsertFilter("/api/v1/pay/unifiedorder", web.BeforeRouter, middleware.JWTAuth)
	web.Router("/api/v1/pay/unifiedorder", &controllers.PaymentController{}, "post:UnifiedOrder")
	// 发起退款（需登录，订单租客/房东）
	web.InsertFilter("/api/v1/pay/refund", web.BeforeRouter, middleware.JWTAuth)
	web.Router("/api/v1/pay/refund", &controllers.PaymentController{}, "post:Refund")
	// 微信支付/退款回调（公开，内部做签名校验）
	web.Router("/api/v1/pay/notify", &controllers.PaymentController{}, "post:Notify")
}
