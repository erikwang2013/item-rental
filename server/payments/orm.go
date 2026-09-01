// 支付模块 ORM 辅助
package payments

import (
	"github.com/beego/beego/v2/client/orm"
)

// ormer 返回默认数据库连接
func ormer() orm.Ormer {
	return orm.NewOrm()
}
