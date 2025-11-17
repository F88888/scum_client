package client

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"qq_client/global"
	"qq_client/internal/websocket_client"
	"qq_client/model/request"
	"qq_client/util"
	"sync"
	"time"
)

// Client represents the SCUM Client
type Client struct {
	config   *global.Config
	wsClient *websocket_client.Client
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// Message types for WebSocket communication
const (
	MsgTypeAuth         = "client_auth"      // 与后端保持一致
	MsgTypeHeartbeat    = "client_heartbeat" // 与后端保持一致
	MsgTypeClientUpdate = "client_update"
	MsgTypeClientStatus = "client_status"
)

// New creates a new SCUM Client
func New(cfg *global.Config) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}

	return client
}

// Start starts the client
func (c *Client) Start() error {
	// Connect to WebSocket server
	u, err := url.Parse(c.config.ServerUrl)
	if err != nil {
		return fmt.Errorf("invalid server address: %w", err)
	}

	// 转换为WebSocket URL
	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
	u.Path = "/api/v1/scum_client/ws"

	// 创建WebSocket客户端（使用简化的logger）
	wsClient := websocket_client.New(u.String(), nil)

	// 设置重连回调
	wsClient.SetCallbacks(
		func() {
			// 连接成功后自动发送认证
			fmt.Printf("[Client] WebSocket连接成功，正在发送认证消息...\n")
			authMsg := request.WebSocketMessage{
				Type: MsgTypeAuth,
				Data: map[string]interface{}{
					"server_id": c.config.ServerID,
					"token":     "scum_client_token", // 这里应该使用实际的token
				},
			}
			fmt.Printf("[Client] 发送认证消息: Type=%s, ServerID=%d\n", authMsg.Type, c.config.ServerID)
			if err = wsClient.SendMessage(authMsg); err != nil {
				fmt.Printf("[Client] 发送认证消息失败: %v\n", err)
			} else {
				fmt.Printf("[Client] 认证消息发送成功\n")
			}
		},
		func() {
			fmt.Printf("[Client] WebSocket连接断开\n")
		},
		func() {
			// 重连成功后重新发送认证
			fmt.Printf("[Client] WebSocket重连成功，正在重新发送认证消息...\n")
			authMsg := request.WebSocketMessage{
				Type: MsgTypeAuth,
				Data: map[string]interface{}{
					"server_id": c.config.ServerID,
					"token":     "scum_client_token", // 这里应该使用实际的token
				},
			}
			fmt.Printf("[Client] 重新发送认证消息: Type=%s, ServerID=%d\n", authMsg.Type, c.config.ServerID)
			if err = wsClient.SendMessage(authMsg); err != nil {
				fmt.Printf("[Client] 重新发送认证消息失败: %v\n", err)
			} else {
				fmt.Printf("[Client] 重新认证消息发送成功\n")
			}
		},
	)

	// 使用自动重连连接
	if err = wsClient.ConnectWithAutoReconnect(); err != nil {
		return fmt.Errorf("failed to connect to WebSocket server: %w", err)
	}

	c.wsClient = wsClient

	// Start message handler
	c.wg.Add(1)
	go c.handleMessages()

	return nil
}

// Stop stops the client
func (c *Client) Stop() {
	fmt.Println("Stopping SCUM Client...")

	c.cancel()

	if c.wsClient != nil {
		if err := c.wsClient.Close(); err != nil {
			fmt.Printf("Failed to close WebSocket client: %v\n", err)
		}
	}

	c.wg.Wait()
}

// handleMessages handles incoming WebSocket messages
func (c *Client) handleMessages() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			// 检查WebSocket客户端是否仍然连接
			if !c.wsClient.IsConnected() {
				time.Sleep(2 * time.Second)
				continue
			}

			var msg request.WebSocketMessage
			if err := c.wsClient.ReadMessage(&msg); err != nil {
				// 连接断开，等待重连
				time.Sleep(2 * time.Second)
				continue
			}

			c.handleMessage(msg)
		}
	}
}

// handleMessage handles a single WebSocket message
func (c *Client) handleMessage(msg request.WebSocketMessage) {
	fmt.Printf("[Client] 收到WebSocket消息: Type=%s, Success=%v\n", msg.Type, msg.Success)

	switch msg.Type {
	case MsgTypeAuth:
		c.handleAuthResponse(msg)
	case MsgTypeHeartbeat:
		// Heartbeat messages from server are handled silently
		fmt.Printf("[Client] 收到心跳消息，正在响应...\n")
		c.handleHeartbeat(msg)
	case MsgTypeClientUpdate:
		c.handleClientUpdate(msg.Data)
	default:
		fmt.Printf("[Client] 未知消息类型: %s\n", msg.Type)
	}
}

// handleAuthResponse handles authentication response
func (c *Client) handleAuthResponse(msg request.WebSocketMessage) {
	fmt.Printf("[Client] 收到认证响应: Success=%v, Error=%s\n", msg.Success, msg.Error)
	if msg.Success {
		fmt.Printf("[Client] 认证成功！\n")

		// 从响应中获取服务器类型并保存到配置
		if data, ok := msg.Data.(map[string]interface{}); ok {
			if ftpProvider, ok := data["ftp_provider"].(float64); ok {
				c.config.FtpProvider = int(ftpProvider)
				fmt.Printf("[Client] 服务器FTP提供商类型已保存: %d\n", c.config.FtpProvider)
			}
		}
	} else {
		fmt.Printf("[Client] 认证失败: %s\n", msg.Error)
	}
}

// handleHeartbeat handles heartbeat message
func (c *Client) handleHeartbeat(msg request.WebSocketMessage) {
	// 回应心跳
	response := request.WebSocketMessage{
		Type:    MsgTypeHeartbeat,
		Success: true,
		Data: map[string]interface{}{
			"timestamp": time.Now().Unix(),
		},
	}
	c.wsClient.SendMessage(response)
}

// handleClientUpdate handles client update request
func (c *Client) handleClientUpdate(data interface{}) {
	fmt.Println("Received update request")

	updateData, ok := data.(map[string]interface{})
	if !ok {
		fmt.Println("Invalid update request data")
		return
	}

	action, _ := updateData["action"].(string)
	updateType, _ := updateData["type"].(string)

	if action == "update" && updateType == "self_update" {
		fmt.Println("Starting self-update process...")

		// 发送更新开始状态
		c.sendResponse(MsgTypeClientUpdate, map[string]interface{}{
			"type":   "self_update",
			"status": "starting",
		}, "")

		// 启动自我更新流程
		go c.performSelfUpdate(updateData)
	}
}

// performSelfUpdate performs the self-update process
func (c *Client) performSelfUpdate(updateData map[string]interface{}) {
	fmt.Println("Performing self-update...")

	// 发送更新状态
	c.sendResponse(MsgTypeClientUpdate, map[string]interface{}{
		"type":    "self_update",
		"status":  "checking",
		"message": "Checking for updates...",
	}, "")

	// 获取下载链接
	downloadURL, _ := updateData["download_url"].(string)

	if downloadURL == "" {
		c.sendResponse(MsgTypeClientUpdate, map[string]interface{}{
			"type":    "self_update",
			"status":  "no_update",
			"message": "No download URL provided",
		}, "")
		return
	}

	// 准备外部更新器
	currentExe, err := os.Executable()
	if err != nil {
		fmt.Printf("Failed to get executable path: %v\n", err)
		c.sendResponse(MsgTypeClientUpdate, map[string]interface{}{
			"type":    "self_update",
			"status":  "failed",
			"message": fmt.Sprintf("Failed to get executable path: %v", err),
		}, "")
		return
	}

	updateConfig := util.ExternalUpdaterConfig{
		CurrentExePath: currentExe,
		UpdateURL:      downloadURL,
		Args:           []string{}, // 可以从配置中获取启动参数
	}

	// 发送更新状态并启动外部更新器
	c.sendResponse(MsgTypeClientUpdate, map[string]interface{}{
		"type":    "self_update",
		"status":  "downloading",
		"message": "Starting external updater...",
	}, "")

	// 启动外部更新器
	if err := util.ExecuteExternalUpdate(updateConfig); err != nil {
		fmt.Printf("Failed to start external updater: %v\n", err)
		c.sendResponse(MsgTypeClientUpdate, map[string]interface{}{
			"type":    "self_update",
			"status":  "failed",
			"message": fmt.Sprintf("Failed to start updater: %v", err),
		}, "")
		return
	}

	fmt.Println("External updater started, shutting down current process...")

	// 发送最终状态
	c.sendResponse(MsgTypeClientUpdate, map[string]interface{}{
		"type":    "self_update",
		"status":  "installing",
		"message": "Updater started, shutting down for update...",
	}, "")

	// 延迟一段时间让消息发送完成，然后退出让更新器接管
	go func() {
		time.Sleep(2 * time.Second)
		fmt.Println("Exiting for update...")
		// 这里应该优雅退出，但为了简化，直接退出
	}()
}

// sendResponse sends a response message
func (c *Client) sendResponse(msgType string, data interface{}, errorMsg string) {
	msg := request.WebSocketMessage{
		Type:    msgType,
		Data:    data,
		Success: errorMsg == "",
		Error:   errorMsg,
	}

	if err := c.wsClient.SendMessage(msg); err != nil {
		fmt.Printf("Failed to send response: %v\n", err)
	}
}
