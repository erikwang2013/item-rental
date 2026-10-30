// 路由注册：挂载 JWT 鉴权中间件与各业务路由
package routers

import (
	"log"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
	"github.com/beego/beego/v2/server/web/filter/cors"
	"github.com/erikwang2013/item-rental/server/controllers"
	"github.com/erikwang2013/item-rental/server/middleware"
	"github.com/erikwang2013/item-rental/server/models"
	"github.com/erikwang2013/item-rental/server/services"
)

// requireAuth 注册"方法感知"的 JWT 过滤器。
// Beego 的 InsertFilter 只按 URL 匹配、与 HTTP 方法无关：若直接给 /api/v1/items/:id
// 挂 JWTAuth，会把同路径下公开的 GET 读接口（物品详情）一并拦截。
// 这里按方法分流：GET（公开读）放行，其余方法（写操作）需通过 JWT 校验。
// 注：当前公开 GET 仅物品列表/详情与类目；订单/用户资料仍挂全局 JWTAuth。
func requireAuth(path string) {
	web.InsertFilter(path, web.BeforeRouter, func(ctx *context.Context) {
		if ctx.Input.Method() == "GET" {
			return
		}
		middleware.JWTAuth(ctx)
	})
}

func init() {
	// 初始化数据库连接与模型
	models.InitORM()

	// 待支付订单超时清理：每 5 分钟把超过支付窗口的订单置为已取消。
	// 0 单+成功支付流水是退款/回调处理中（退款接口先抢占订单再调网关），
	// 不能按超时取消，否则退款重试会被 409 阻断；排除这类订单。
	// 条件更新幂等，多实例并发安全；goroutine 不阻塞启动。
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			cutoff := time.Now().Add(-services.PendingOrderHold)
			if _, err := orm.NewOrm().Raw(
				`UPDATE orders o SET o.status=?, o.cancel_reason=?
				 WHERE o.status=? AND o.created_at<?
				   AND NOT EXISTS (SELECT 1 FROM payments p WHERE p.order_id=o.id AND p.status=?)`,
				models.OrderStatusCancelled, "超时未支付", models.OrderStatusPending, cutoff, models.PaymentStatusSuccess,
			).Exec(); err != nil {
				log.Printf("清理超时未支付订单失败: %v", err)
			}
		}
	}()

	// 全局 CORS(先于安全过滤器注册;dev 默认 *,app.conf cors_allow_origins 可收紧)
	// OPTIONS 预检在此放行
	web.InsertFilter("/*", web.BeforeRouter, cors.Allow(&cors.Options{
		AllowOrigins: []string{web.AppConfig.DefaultString("cors_allow_origins", "*")},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Authorization", "Content-Type"},
		MaxAge:       600,
	}))

	// 健康检查（公开）
	web.Get("/health", func(ctx *context.Context) {
		_ = ctx.Output.JSON(map[string]any{"code": 0, "msg": "ok"}, false, false)
	})

	// 认证相关（公开）
	web.Router("/api/v1/auth/login", &controllers.AuthController{}, "post:Login")
	web.Router("/api/v1/auth/refresh", &controllers.AuthController{}, "post:Refresh")
	web.InsertFilter("/api/v1/auth/logout", web.BeforeRouter, middleware.JWTAuth)
	web.Router("/api/v1/auth/logout", &controllers.AuthController{}, "post:Logout")

	// 用户信息（需登录；GET 也非公开——仅本人可见自己的资料）
	web.InsertFilter("/api/v1/user/profile", web.BeforeRouter, middleware.JWTAuth)
	web.Router("/api/v1/user/profile", &controllers.UserController{}, "get:Profile")
	web.Router("/api/v1/user/profile", &controllers.UserController{}, "put:UpdateProfile")

	// 头像上传（需登录）
	web.InsertFilter("/api/v1/user/avatar", web.BeforeRouter, middleware.JWTAuth)
	web.Router("/api/v1/user/avatar", &controllers.UserController{}, "post:UploadAvatar")

	// 品类（公开读取）
	web.Router("/api/v1/categories", &controllers.CategoryController{}, "get:List")

	// 物品（公开读取）
	web.Router("/api/v1/items", &controllers.ItemController{}, "get:List")
	web.Router("/api/v1/items/:id", &controllers.ItemController{}, "get:Detail")

	// 物品写操作（需登录：发布/更新/下架）；GET 读公开
	requireAuth("/api/v1/items")
	requireAuth("/api/v1/items/:id")
	requireAuth("/api/v1/items/:id/offshelf")
	requireAuth("/api/v1/items/upload")
	web.Router("/api/v1/items", &controllers.ItemController{}, "post:Create")
	web.Router("/api/v1/items/:id", &controllers.ItemController{}, "put:Update")
	web.Router("/api/v1/items/:id/offshelf", &controllers.ItemController{}, "post:OffShelf")
	web.Router("/api/v1/items/upload", &controllers.ItemController{}, "post:Upload")

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
