package util

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"qq_client/global"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketClient WebSocket客户端
type WebSocketClient struct {
	conn      *websocket.Conn
	url       string
	token     string
	serverID  uint
	isRunning bool
	mutex     sync.RWMutex
	stopChan  chan struct{}
	// 重连配置
	maxRetries       int
	retryInterval    time.Duration
	maxRetryInterval time.Duration
	// 心跳配置
	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration
	lastHeartbeat     time.Time
	// 回调函数
	onConnect    func()
	onDisconnect func()
	onReconnect  func()
}

// WebSocketMessage WebSocket消息结构
type WebSocketMessage struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Success bool        `json:"success"`
}

// 消息类型
const (
	MsgTypeClientAuth      = "auth"
	MsgTypeClientHeartbeat = "heartbeat"
	MsgTypeClientUpdate    = "client_update"
	MsgTypeClientStatus    = "client_status"
	MsgTypeClientResponse  = "client_response"
)

// NewWebSocketClient 创建新的WebSocket客户端
func NewWebSocketClient(serverURL string, serverID uint) *WebSocketClient {
	return &WebSocketClient{
		url:               serverURL,
		serverID:          serverID,
		stopChan:          make(chan struct{}),
		maxRetries:        -1,               // 无限重试
		retryInterval:     3 * time.Second,  // 减少重连间隔，快速恢复
		maxRetryInterval:  30 * time.Second, // 减少最大重连间隔
		heartbeatInterval: 30 * time.Second, // 心跳间隔
		heartbeatTimeout:  30 * time.Minute, // 大幅延长心跳超时到30分钟，避免误判断开
	}
}

// Connect 连接到WebSocket服务器
func (c *WebSocketClient) Connect() error {
	// 构建WebSocket URL
	u, err := url.Parse(c.url)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}

	// 转换为WebSocket URL
	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
	u.Path = "/api/v1/scum_client/ws"

	log.Printf("Connecting to WebSocket: %s", u.String())

	// 配置WebSocket拨号器，优化连接稳定性
	dialer := websocket.Dialer{
		HandshakeTimeout:  60 * time.Second, // 延长握手超时时间到60秒
		ReadBufferSize:    128 * 1024,       // 128KB 读取缓冲区
		WriteBufferSize:   128 * 1024,       // 128KB 写入缓冲区
		EnableCompression: false,            // 禁用压缩减少CPU开销
	}

	// 连接WebSocket
	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	// 设置连接参数 - 移除超时限制提高稳定性
	conn.SetReadLimit(2 * 1024 * 1024) // 2MB 最大消息大小
	// 移除读取和写入超时限制，避免网络波动导致的连接断开
	// conn.SetReadDeadline() - 不设置读取超时
	// conn.SetWriteDeadline() - 不设置写超时

	// 设置Pong处理器，但不设置读取超时
	conn.SetPongHandler(func(string) error {
		// 只更新最后心跳时间，不重新设置读取超时
		c.mutex.Lock()
		c.lastHeartbeat = time.Now()
		c.mutex.Unlock()
		return nil
	})

	c.mutex.Lock()
	c.conn = conn
	c.isRunning = true
	c.mutex.Unlock()

	// 发送认证消息
	if err := c.authenticate(); err != nil {
		c.Close()
		return fmt.Errorf("authentication failed: %w", err)
	}

	// 启动消息处理循环
	go c.messageLoop()
	go c.heartbeatLoop()

	log.Printf("WebSocket client connected and authenticated")

	// 调用连接回调
	if c.onConnect != nil {
		c.onConnect()
	}

	return nil
}

// authenticate 认证
func (c *WebSocketClient) authenticate() error {
	authMsg := WebSocketMessage{
		Type: MsgTypeClientAuth,
		Data: map[string]interface{}{
			"server_id": c.serverID,
			"token":     "scum_client_token", // 这里应该使用实际的token
		},
	}

	return c.SendMessage(authMsg)
}

// SendMessage 发送消息
func (c *WebSocketClient) SendMessage(msg WebSocketMessage) error {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.isRunning || c.conn == nil {
		return fmt.Errorf("WebSocket client is not connected")
	}

	// 移除写超时限制，避免网络波动时发送失败
	// c.conn.SetWriteDeadline() - 不设置写超时

	return c.conn.WriteJSON(msg)
}

// messageLoop 消息处理循环
func (c *WebSocketClient) messageLoop() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("WebSocket message loop panic: %v", r)
		}
	}()

	for {
		select {
		case <-c.stopChan:
			return
		default:
			c.mutex.RLock()
			if !c.isRunning || c.conn == nil {
				c.mutex.RUnlock()
				return
			}
			conn := c.conn
			c.mutex.RUnlock()

			var msg WebSocketMessage
			err := conn.ReadJSON(&msg)
			if err != nil {
				log.Printf("WebSocket read error: %v", err)
				c.handleDisconnection()
				return
			}

			c.handleMessage(msg)
		}
	}
}

// handleMessage 处理接收到的消息
func (c *WebSocketClient) handleMessage(msg WebSocketMessage) {
	log.Printf("Received WebSocket message: %s", msg.Type)

	switch msg.Type {
	case MsgTypeClientAuth:
		c.handleAuthResponse(msg)
	case MsgTypeClientHeartbeat:
		c.handleHeartbeat(msg)
	case MsgTypeClientUpdate:
		c.handleUpdateRequest(msg)
	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

// handleAuthResponse 处理认证响应
func (c *WebSocketClient) handleAuthResponse(msg WebSocketMessage) {
	if msg.Success {
		log.Printf("Authentication successful")

		// 从响应中获取服务器类型并保存到配置
		if data, ok := msg.Data.(map[string]interface{}); ok {
			if ftpProvider, ok := data["ftp_provider"].(float64); ok {
				global.ScumConfig.FtpProvider = int(ftpProvider)
				log.Printf("Server FTP Provider type saved: %d", global.ScumConfig.FtpProvider)
			}
		}
	} else {
		log.Printf("Authentication failed: %s", msg.Error)
	}
}

// handleHeartbeat 处理心跳
func (c *WebSocketClient) handleHeartbeat(msg WebSocketMessage) {
	// 回应心跳
	response := WebSocketMessage{
		Type:    MsgTypeClientHeartbeat,
		Success: true,
		Data: map[string]interface{}{
			"timestamp": time.Now().Unix(),
		},
	}
	c.SendMessage(response)
}

// handleUpdateRequest 处理更新请求
func (c *WebSocketClient) handleUpdateRequest(msg WebSocketMessage) {
	log.Printf("Received update request")

	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		log.Printf("Invalid update request data")
		return
	}

	action, _ := data["action"].(string)
	updateType, _ := data["type"].(string)

	if action == "update" && updateType == "self_update" {
		log.Printf("Starting self-update process...")

		// 发送更新开始状态
		c.SendMessage(WebSocketMessage{
			Type:    MsgTypeClientUpdate,
			Success: true,
			Data: map[string]interface{}{
				"type":   "self_update",
				"status": "starting",
			},
		})

		// 启动自我更新流程
		go c.performSelfUpdate()
	}
}

// performSelfUpdate 执行自我更新
func (c *WebSocketClient) performSelfUpdate() {
	log.Printf("Performing self-update...")

	// 发送更新状态
	c.SendMessage(WebSocketMessage{
		Type:    MsgTypeClientUpdate,
		Success: true,
		Data: map[string]interface{}{
			"type":    "self_update",
			"status":  "checking",
			"message": "Checking for updates...",
		},
	})

	// 1. 检查更新（这里应该实现实际的检查逻辑）
	// 暂时模拟有更新可用
	updateAvailable := false // 设置为false表示暂时没有更新
	downloadURL := "https://github.com/your-org/scum_client/releases/download/latest/scum_client.exe"

	if !updateAvailable {
		c.SendMessage(WebSocketMessage{
			Type:    MsgTypeClientUpdate,
			Success: true,
			Data: map[string]interface{}{
				"type":    "self_update",
				"status":  "no_update",
				"message": "No updates available",
			},
		})
		return
	}

	// 2. 准备外部更新器
	currentExe, err := os.Executable()
	if err != nil {
		log.Printf("Failed to get executable path: %v", err)
		c.SendMessage(WebSocketMessage{
			Type:    MsgTypeClientUpdate,
			Success: false,
			Data: map[string]interface{}{
				"type":    "self_update",
				"status":  "failed",
				"message": fmt.Sprintf("Failed to get executable path: %v", err),
			},
		})
		return
	}

	updateConfig := ExternalUpdaterConfig{
		CurrentExePath: currentExe,
		UpdateURL:      downloadURL,
		Args:           os.Args[1:], // 排除程序名本身
	}

	// 3. 发送更新状态并启动外部更新器
	c.SendMessage(WebSocketMessage{
		Type:    MsgTypeClientUpdate,
		Success: true,
		Data: map[string]interface{}{
			"type":    "self_update",
			"status":  "downloading",
			"message": "Starting external updater...",
		},
	})

	// 启动外部更新器
	if err := ExecuteExternalUpdate(updateConfig); err != nil {
		log.Printf("Failed to start external updater: %v", err)
		c.SendMessage(WebSocketMessage{
			Type:    MsgTypeClientUpdate,
			Success: false,
			Data: map[string]interface{}{
				"type":    "self_update",
				"status":  "failed",
				"message": fmt.Sprintf("Failed to start updater: %v", err),
			},
		})
		return
	}

	log.Printf("External updater started, shutting down current process...")

	// 发送最终状态
	c.SendMessage(WebSocketMessage{
		Type:    MsgTypeClientUpdate,
		Success: true,
		Data: map[string]interface{}{
			"type":    "self_update",
			"status":  "installing",
			"message": "Updater started, shutting down for update...",
		},
	})

	// 延迟一段时间让消息发送完成，然后退出让更新器接管
	go func() {
		time.Sleep(2 * time.Second)
		log.Printf("Exiting for update...")
		os.Exit(0)
	}()
}

// heartbeatLoop 心跳循环
func (c *WebSocketClient) heartbeatLoop() {
	ticker := time.NewTicker(c.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			if c.IsConnected() {
				c.sendHeartbeat()
			}
		}
	}
}

// sendHeartbeat 发送心跳
func (c *WebSocketClient) sendHeartbeat() {
	heartbeatMsg := WebSocketMessage{
		Type:    MsgTypeClientHeartbeat,
		Success: true,
		Data: map[string]interface{}{
			"timestamp": time.Now().Unix(),
		},
	}

	if err := c.SendMessage(heartbeatMsg); err != nil {
		log.Printf("Failed to send heartbeat: %v", err)
		c.handleDisconnection()
	} else {
		c.mutex.Lock()
		c.lastHeartbeat = time.Now()
		c.mutex.Unlock()
	}
}

// handleDisconnection 处理断线
func (c *WebSocketClient) handleDisconnection() {
	c.mutex.Lock()
	c.isRunning = false
	c.mutex.Unlock()

	log.Printf("WebSocket disconnected, attempting to reconnect...")

	// 尝试重连
	go c.reconnect()
}

// reconnect 重连
func (c *WebSocketClient) reconnect() {
	backoff := c.retryInterval
	retryCount := 0

	for {
		select {
		case <-c.stopChan:
			return
		case <-time.After(backoff):
			log.Printf("Attempting to reconnect... (attempt %d)", retryCount+1)

			if err := c.Connect(); err != nil {
				log.Printf("Reconnection failed: %v", err)
				retryCount++

				// 检查是否达到最大重试次数
				if c.maxRetries > 0 && retryCount >= c.maxRetries {
					log.Printf("Max retry attempts reached, giving up")
					return
				}

				// 指数退避
				backoff *= 2
				if backoff > c.maxRetryInterval {
					backoff = c.maxRetryInterval
				}
			} else {
				log.Printf("Reconnected successfully")

				// 调用重连回调
				if c.onReconnect != nil {
					c.onReconnect()
				}

				return
			}
		}
	}
}

// Close 关闭连接
func (c *WebSocketClient) Close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.isRunning {
		c.isRunning = false
		close(c.stopChan)

		if c.conn != nil {
			c.conn.Close()
		}

		// 调用断开连接回调
		if c.onDisconnect != nil {
			c.onDisconnect()
		}
	}

	log.Printf("WebSocket client closed")
}

// IsConnected 检查是否已连接
func (c *WebSocketClient) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.isRunning && c.conn != nil
}

// SetCallbacks 设置回调函数
func (c *WebSocketClient) SetCallbacks(onConnect, onDisconnect, onReconnect func()) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.onConnect = onConnect
	c.onDisconnect = onDisconnect
	c.onReconnect = onReconnect
}

// SetRetryConfig 设置重试配置
func (c *WebSocketClient) SetRetryConfig(maxRetries int, retryInterval, maxRetryInterval time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.maxRetries = maxRetries
	c.retryInterval = retryInterval
	c.maxRetryInterval = maxRetryInterval
}

// SetHeartbeatConfig 设置心跳配置
func (c *WebSocketClient) SetHeartbeatConfig(interval, timeout time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.heartbeatInterval = interval
	c.heartbeatTimeout = timeout
}
