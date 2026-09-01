// 多端租赁平台 —— 后端入口
// beego v2 + security-go 安全防护 + go-scout 搜索同步
package main

import (
	"github.com/beego/beego/v2/server/web"

	_ "github.com/erikwang2013/item-rental/server/routers"
)

func main() {
	web.Run()
}
