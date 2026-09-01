// 控制器基类：统一响应格式
package controllers

import (
	"github.com/beego/beego/v2/server/web"
)

// BaseController 提供统一 JSON 响应方法
type BaseController struct {
	web.Controller
}

// OK 成功响应
func (c *BaseController) OK(data any) {
	c.Data["json"] = map[string]any{"code": 0, "msg": "ok", "data": data}
	c.ServeJSON()
}

// Fail 失败响应
func (c *BaseController) Fail(code int, msg string) {
	c.Data["json"] = map[string]any{"code": code, "msg": msg}
	c.ServeJSON()
}
