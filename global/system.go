package global

import "regexp"

type Config struct {
	ServerID  uint   `json:"server_id" yaml:"server_id"`
	ServerUrl string `json:"server_url" yaml:"server_url"`
}

var (
	ScumConfig            Config
	ExtractLocationRegexp = regexp.MustCompile("^(.*) Location \"{X=\\d+(\\.\\d+)? Y=\\d+(\\.\\d+)? Z=\\d+(\\.\\d+)?}\"-(\\d{1,10})$")
)
