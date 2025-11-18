package _const

// 聊天模式颜色常量（十六进制格式）
const (
	// ChatColorLocal LOCAL 聊天模式的颜色值
	ChatColorLocal = "404347"
	// ChatColorGlobal GLOBAL 聊天模式的颜色值
	ChatColorGlobal = "183842"
	// ChatColorAdmin ADMIN 聊天模式的颜色值
	ChatColorAdmin = "3D4325"
)

// 颜色匹配相关常量
const (
	// ColorMatchThreshold 颜色匹配阈值（RGB空间中的欧几里得距离）
	// 当两个颜色的距离小于此值时，认为颜色接近
	ColorMatchThreshold = 30.0
)
