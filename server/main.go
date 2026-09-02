// 多端租赁平台 —— 后端入口
// beego v2 + security-go 安全防护 + go-scout 搜索同步
package main

import (
	"github.com/beego/beego/v2/server/web"

	_ "github.com/erikwang2013/item-rental/server/routers"
)

// 吉祥物「租租龟」启动横幅 —— 信/借/还
const mascotBanner = `
     ,,,
    (o o)  🐢 租租龟
   /=====\  🗝 租赁·就爱有人借
    ~~~~
`

func main() {
	// 静态资源(吉祥物 etc.)→ GET /static/mascot.svg
	web.SetStaticPath("/static", "static")
	web.BConfig.Log.AccessLogs = false
	print(mascotBanner)
	web.Run()
}
