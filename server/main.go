// 多端租赁平台 —— 后端入口
// beego v2
package main

import (
	"github.com/beego/beego/v2/server/web"

	"github.com/erikwang2013/item-rental/server/payments"
	_ "github.com/erikwang2013/item-rental/server/routers"
	"github.com/erikwang2013/item-rental/server/services"
)

// 吉祥物「租租鼠」启动横幅 —— Go 官方 Gopher · 按时借、准时还
const mascotBanner = `
     ____
    /    \\
   |  ••  |  🐹 租租鼠 (Go Gopher)
    \\____/
   ╭─╮  ╭─╮   👷 安全帽 · 📦 抱箱 · ¥ 徽章
    ╰─╯  ╰─╯
        按时借 · 准时还
`

func main() {
	// 请求体上限：普通 JSON 1MB；上传 40MB（9 张 4MB 图片 + multipart 开销）
	web.BConfig.MaxMemory = 1 << 20
	web.BConfig.MaxUploadSize = 40 << 20

	// 启动即校验支付配置与 PII 密钥：空密钥运行会让支付回调可伪造、PII 无法加密
	if err := payments.ValidateConfig(payments.LoadConfig()); err != nil {
		panic("支付配置非法: " + err.Error())
	}
	_ = services.PiiKey() // 缺失/非法密钥在此 panic，避免首次登录才失败

	// 静态资源(吉祥物 etc.)→ GET /static/mascot.svg
	web.SetStaticPath("/static", "static")
	web.BConfig.Log.AccessLogs = false
	print(mascotBanner)
	web.Run()
}
