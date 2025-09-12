package main

import (
	"embed"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"qq_client/global"
	"qq_client/server"
	"qq_client/util"
)

//go:embed config.yaml assets/ocr_setup.bat assets/ocr_setup_simple.bat assets/ocr_server.py assets/download_model.py assets/check_models.py assets/fix_ocr_models.bat
var File embed.FS

// extractEmbeddedFiles 提取嵌入的文件到当前目录
func extractEmbeddedFiles() error {
	// 文件映射：嵌入路径 -> 输出文件名
	fileMap := map[string]string{
		"assets/ocr_setup.bat":        "ocr_setup.bat",
		"assets/ocr_setup_simple.bat": "ocr_setup_simple.bat",
		"assets/ocr_server.py":        "ocr_server.py",
		"assets/download_model.py":    "download_model.py",
		"assets/check_models.py":      "check_models.py",
		"assets/fix_ocr_models.bat":   "fix_ocr_models.bat",
	}

	for embeddedPath, outputFileName := range fileMap {
		// 检查文件是否已存在
		if _, err := os.Stat(outputFileName); !os.IsNotExist(err) {
			fmt.Printf("文件 %s 已存在，跳过提取\n", outputFileName)
			continue
		}

		// 从嵌入文件系统中读取文件内容
		content, err := File.ReadFile(embeddedPath)
		if err != nil {
			return fmt.Errorf("读取嵌入文件 %s 失败: %v", embeddedPath, err)
		}

		// 写入到当前目录
		err = os.WriteFile(outputFileName, content, 0644)
		if err != nil {
			return fmt.Errorf("写入文件 %s 失败: %v", outputFileName, err)
		}

		fmt.Printf("已提取文件: %s\n", outputFileName)
	}

	return nil
}

func main() {
	// init
	var err error

	// 首先提取嵌入的 OCR 相关文件
	fmt.Println("正在提取 OCR 必需文件...")
	if err = extractEmbeddedFiles(); err != nil {
		fmt.Printf("提取 OCR 文件失败: %v\n", err)
		fmt.Println("程序将继续运行，但 OCR 功能可能不可用")
	}

	// 确保 OCR 服务运行
	fmt.Println("检查 OCR 服务状态...")
	if err = util.EnsureOCRService(); err != nil {
		fmt.Printf("OCR 服务启动失败: %v\n", err)
		fmt.Println("程序将继续运行，但图片识别功能可能不可用")
		fmt.Println("请手动运行 ocr_setup.bat 来设置 OCR 环境")
	} else {
		fmt.Println("OCR 服务已就绪")

		// 加载配置文件
		var configData []byte
		var err error

		// 首先尝试从嵌入文件加载
		if configData, err = File.ReadFile("config.yaml"); err != nil {
			fmt.Printf("无法从嵌入文件加载配置: %v\n", err)

			// 尝试从外部文件加载
			if configData, err = os.ReadFile("config.yaml"); err != nil {
				fmt.Printf("无法从外部文件加载配置: %v\n", err)
				fmt.Println("程序将退出，请确保配置文件存在")
				return
			}
			fmt.Println("从外部文件加载配置成功")
		} else {
			fmt.Println("从嵌入文件加载配置成功")
		}

		// 解析配置文件
		if err = yaml.Unmarshal(configData, &global.ScumConfig); err != nil {
			fmt.Printf("解析配置文件失败: %v\n", err)
			return
		}

		fmt.Printf("配置加载成功 - ServerID: %d, ServerUrl: %s\n", global.ScumConfig.ServerID, global.ScumConfig.ServerUrl)

		// 循环机器人主逻辑
		for {
			server.Start()
		}
	}
}
