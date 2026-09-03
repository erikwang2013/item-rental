// snowflake 主键生成：所有业务表(users/items/orders/payments/deposits/messages/credit_events)
// 主键从 MySQL AUTO_INCREMENT 切换为 snowflake id(bwmarrin/snowflake)。
// JSON 输出统一 `,string`：id > 2^53 超出 JS Number 安全整数,客户端须按字符串处理。
package services

import (
	"sync"

	"github.com/beego/beego/v2/server/web"
	"github.com/bwmarrin/snowflake"
)

var (
	sfOnce sync.Once
	sfNode *snowflake.Node
)

// initSnowflake 按 app.conf snowflake_node(默认 1)初始化。
// 多实例部署须每实例唯一(0-1023),否则同毫秒序列位会碰撞。
func initSnowflake() {
	nodeID := int64(web.AppConfig.DefaultInt("snowflake_node", 1))
	n, err := snowflake.NewNode(nodeID)
	if err != nil {
		panic("snowflake 节点号非法: " + err.Error())
	}
	sfNode = n
}

// NextID 生成下一主键(并发安全,进程内单调递增)。
func NextID() int64 {
	sfOnce.Do(initSnowflake)
	return sfNode.Generate().Int64()
}
