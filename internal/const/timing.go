package _const

import "time"

// 时间相关常量
const (
	// 等待时间常量
	DefaultWaitTime = 2 * time.Second // 默认等待时间
	LongWaitTime    = 5 * time.Second // 长时间等待
	ShortWaitTime   = 1 * time.Second // 短时间等待

	// 重试相关常量
	ClientRetryCount = 5 // 客户端重试次数

	// 连接相关常量
	ConnectionTimeout = 60 * time.Second // 连接超时时间
	HeartbeatInterval = 40 * time.Second // 心跳间隔
	HeartbeatTimeout  = 5 * time.Minute  // 心跳超时
	RetryInterval     = 5 * time.Second  // 重试间隔
	MaxRetryInterval  = 60 * time.Second // 最大重试间隔

	// 缓冲区大小常量
	ReadBufferSize  = 128 * 1024      // 读取缓冲区大小
	WriteBufferSize = 128 * 1024      // 写入缓冲区大小
	MaxMessageSize  = 2 * 1024 * 1024 // 最大消息大小

	// OCR 服务相关时间常量
	OCRServiceMaxWaitTime        = 60 * time.Second // OCR 服务最大等待时间
	OCRServicePortCheckTimeout   = 1 * time.Second  // OCR 服务端口检测超时时间
	OCRServiceHealthCheckTimeout = 3 * time.Second  // OCR 服务健康检查超时时间
	OCRServiceRestartWaitTime    = 2 * time.Second  // OCR 服务重启等待时间
	OCRServiceAPITimeout         = 10 * time.Second // OCR 服务 API 请求超时时间
)
