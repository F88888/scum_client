package main

import (
	"embed"
	"gopkg.in/yaml.v3"
	"qq_client/global"
	"qq_client/server"
)

//go:embed config.yaml
var File embed.FS

func main() {
	// init
	var err error
	var config []byte
	// casBin config
	if config, err = File.ReadFile("config.yaml"); err == nil {
		// 解析配置文件
		err = yaml.Unmarshal(config, &global.ScumConfig)
	}
	// 循环机器人主逻辑
	for {
		server.Start()
	}
}
