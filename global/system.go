package global

import "regexp"

type Config struct {
	ServerID    uint   `json:"server_id" yaml:"server_id"`
	ServerUrl   string `json:"server_url" yaml:"server_url"`
	FtpProvider int    `json:"ftp_provider" yaml:"ftp_provider"` // FTP提供商类型: 1=GPORTAL, 2=PingPerfect, 3=自建服务器, 4=命令行服务器
}

// OCRRequest 定义请求结构
type OCRRequest struct {
	Base64  string                 `json:"base64"`
	Options map[string]interface{} `json:"options"`
}

// OCRResponse 定义响应结构
type OCRResponse struct {
	Code int `json:"code"`
	Data []struct {
		Text  string      `json:"text"`
		Score float64     `json:"score"`
		Box   [][]float64 `json:"box"` // 或使用 [][2]float64
		End   string      `json:"end"`
	} `json:"data"`
}

var (
	ScumConfig            Config
	ExtractLocationRegexp = regexp.MustCompile("^(.*) Location \"{X=\\d+(\\.\\d+)? Y=\\d+(\\.\\d+)? Z=\\d+(\\.\\d+)?}\"-(\\d{1,10})$")
)
