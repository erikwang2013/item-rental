// ORM 初始化：注册模型并连接 MySQL
package models

import (
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
	_ "github.com/go-sql-driver/mysql"
)

// InitORM 初始化数据库连接与模型注册。
// 在 routers init 中调用。
func InitORM() {
	sqlconn := web.AppConfig.DefaultString("sqlconn", "")
	if sqlconn == "" {
		panic("app.conf 缺少 sqlconn 数据库配置")
	}

	_ = orm.RegisterDriver("mysql", orm.DRMySQL)
	// 注册全部模型
	orm.RegisterModel(new(User), new(Category), new(Item), new(Order), new(Payment), new(Deposit))

	// 连接数据库（alias "default"）
	err := orm.RegisterDataBase("default", "mysql", sqlconn)
	if err != nil {
		panic("数据库连接失败: " + err.Error())
	}

	// 开发环境自动建表（生产请关闭并改用迁移）
	if web.BConfig.RunMode == "dev" {
		if err := orm.RunSyncdb("default", false, true); err != nil {
			panic("同步数据库表结构失败: " + err.Error())
		}
	}
}
