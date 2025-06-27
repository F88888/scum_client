package main

import (
	"embed"
	"fmt"
	"gopkg.in/y
	"gopkg.in/yaml.v3"
	"os"
	"os/signal"
	"qq_client/global"
	"qq_client/server"
	"qq_client/util"
	"fmt"
)

//go:embed config.yaml
var File embed.FS

func main() {
	// init
	var err error

	

	
	// 设置信号处理，优雅退出
	sigChan := make(chan os.Signal, 1)

	
	// 启动一个goroutine处理信号
	go func() {
		<-sigChan

		
		// 停止 OCR 服务

		
		// 关闭日志文件

		
		fmt.Println("程序已安全退出")
		os.Exit(0)

	
	// 确保 OCR 服务运行
	fmt.Println("检查 OCR 服务状态...")
	if err = util.EnsureOCRService(); err != nil {
		fmt.Printf("OCR 服务启动失败: %v\n", err)
		fmt.Println("程序将继续运行，但图片识别功能可能不可用")
		fmt.Println("请手动运行 ocr_setup.bat 来设置 OCR 环境")
	} else {
		fmt.Println("OCR 服务已就绪")

	
	// casBin config
	if config, err = File.ReadFile("config.yaml"); err == nil {
		// 解析配置文件
		err = yaml.Unmarshal(config, &global.ScumConfig)

	

	
	// 循环机器人主逻辑
	for {
		server.Start()
	}
}
