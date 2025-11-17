package request

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Success bool        `json:"success"`
}

// OcrResult represents a message sent over WebSocket
type OcrResult struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
}
