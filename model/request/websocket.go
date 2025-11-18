package request

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Success bool        `json:"success"`
}

// OcrItem represents a single OCR recognition item
type OcrItem struct {
	Text       string      `json:"text"`
	Confidence float64     `json:"confidence"`
	Box        [][]float64 `json:"box"`      // 四个顶点坐标 [[x1,y1], [x2,y2], [x3,y3], [x4,y4]]
	Position   OcrPosition `json:"position"` // 矩形边界框
}

// OcrPosition represents the bounding box position
type OcrPosition struct {
	Left   int `json:"left"`
	Top    int `json:"top"`
	Right  int `json:"right"`
	Bottom int `json:"bottom"`
}

// OcrResult represents OCR recognition result
type OcrResult struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`    // 合并的完整文字（向后兼容）
	Items   []OcrItem   `json:"items"`   // 识别到的文本块数组（新格式）
	Message string      `json:"message"` // 响应消息
}
