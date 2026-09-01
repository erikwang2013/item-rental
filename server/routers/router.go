// 路由注册：挂载 security-go 安全中间件、JWT 鉴权中间件与各业务路由
package routers

import (
	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
	"github.com/erikwang2013/item-rental/server/controllers"
	"github.com/erikwang2013/item-rental/server/middleware"
	"github.com/erikwang2013/item-rental/server/models"
	"github.com/erikwang2013/item-rental/server/search"
	"github.com/erikwang2013/item-rental/server/security"
)

func init() {
	// 初始化攻击检测引擎（注册全部检测器）
	security.InitEngine()
	// 初始化数据库连接与模型
	models.InitORM()
	// 初始化搜索引擎（go-scout + OpenSearch）
	search.Init()

	// 全局安全过滤器：对每个请求做攻击检测 + IP 自动封禁
	// 置于最前，拦截恶意请求
	web.InsertFilter("/*", web.BeforeRouter, middleware.SecurityFilter)

	// 健康检查（公开）
	web.Get("/health", func(ctx *context.Context) {
		ctx.Output.JSON(map[string]any{"code": 0, "msg": "ok"}, false, false)
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

	// 物品（需登录：发布/更新/下架）
	web.InsertFilter("/api/v1/items/:id", web.BeforeRouter, middleware.JWTAuth)
	web.Router("/api/v1/items", &controllers.ItemController{}, "post:Create")
	web.Router("/api/v1/items/:id", &controllers.ItemController{}, "put:Update")
	web.Router("/api/v1/items/:id/offshelf", &controllers.ItemController{}, "post:OffShelf")

	// ---- M3 微信支付 ----
	// 创建支付单（需登录）
	web.InsertFilter("/api/v1/pay/unifiedorder", web.BeforeRouter, middleware.JWTAuth)
	web.Router("/api/v1/pay/unifiedorder", &controllers.PaymentController{}, "post:UnifiedOrder")
	// 微信支付回调（公开，内部做签名校验）
	web.Router("/api/v1/pay/notify", &controllers.PaymentController{}, "post:Notify")
}
