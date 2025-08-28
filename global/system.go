package global

import "regexp"

type Config struct {
	ServerID  uint   `json:"server_id" yaml:"server_id"`
	ServerUrl string `json:"server_url" yaml:"server_url"`
}

// OCRRequest 定义请求结构
type OCRRequest struct {
	Base64  string                 `json:"base64"`
	Options map[string]interface{} `json:"options"`
}

// OCRResponse 定义响应结构
type OCRResponse struct {
	Code      int     `json:"code"`
	Data      string  `json:"data"`
	Message   string  `json:"message"`
	Time      float64 `json:"time"`
	Timestamp float64 `json:"timestamp"`
}

var (
	ScumConfig            Config
	ExtractLocationRegexp = regexp.MustCompile("^(.*) Location \"{X=\\d+(\\.\\d+)? Y=\\d+(\\.\\d+)? Z=\\d+(\\.\\d+)?}\"-(\\d{1,10})$")
)
