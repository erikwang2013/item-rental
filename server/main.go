// 多端租赁平台 —— 后端入口
// beego v2 + security-go 安全防护 + go-scout 搜索同步
package main

import (
	"github.com/beego/beego/v2/server/web"

	_ "github.com/erikwang2013/item-rental/server/routers"
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
	// 静态资源(吉祥物 etc.)→ GET /static/mascot.svg
	web.SetStaticPath("/static", "static")
	web.BConfig.Log.AccessLogs = false
	print(mascotBanner)
	web.Run()
}
