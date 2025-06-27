package util

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"qq_client/global"
	"strings"
	"time"
)

// ExtractTextFromSpecifiedAreaAndValidateThreeTimes
// @author: [Fantasia](https://www.npc0.com)
// @function: ExtractTextFromSpecifiedAreaAndValidateThreeTimes
// @description: 从指定区域提取文本并验证三次
// @param: x1, y1, x2, y2 int 左上角、右下角坐标
// @param: test string 期望识别的文本
// @return: error
func ExtractTextFromSpecifiedAreaAndValidateThreeTimes(x1, y1, x2, y2 int, test string) error {
	// init
	var i int
	var err error
	var imagePath string

	// 提取图片
	if imagePath, err = ScreenshotGrayscale(x1, y1, x2, y2); err != nil {
		return err
	}
	// 移除图片
	defer func(name string) {
		_ = os.Remove(name)
	}(imagePath)

	for {
		i += 1
		if i > 3 {
			break
		}

		// 提取文本
		var imageData, jsonData []byte
		imageData, err = os.ReadFile(imagePath)
		if err != nil {
			fmt.Printf("读取图片文件失败: %v\n", err)
			continue
		}

		// 将图片转换为Base64编码
		base64Data := base64.StdEncoding.EncodeToString(imageData)

		// 构造请求参数 - 新增target_text参数
		requestData := map[string]interface{}{
			"Base64":      base64Data,
			"target_text": test, // 新增目标文字参数
		}

		// 将请求数据转换为JSON
		jsonData, err = json.Marshal(requestData)
		if err != nil {
			fmt.Printf("JSON编码失败: %v\n", err)
			continue
		}

		// 发送POST请求到PaddleOCR服务
		var resp *http.Response
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err = client.Post("http://127.0.0.1:1224/api/ocr",
			"application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("发送请求失败: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		// 读取并解析响应
		jsonData, err = io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("读取响应失败: %v\n", err)
			continue
		}

		var ocrResponse global.OCRResponse
		err = json.Unmarshal(jsonData, &ocrResponse)
		if err != nil {
			fmt.Printf("解析响应JSON失败: %v\n", err)
			continue
		}

		// 处理响应结果
		if ocrResponse.Code == 100 {
			// 成功找到目标文字
			fmt.Printf("OCR识别成功: 找到目标文字 '%s'\n", test)
			return nil
		} else if ocrResponse.Code == 200 {
			// 识别到文字但不是目标文字
			fmt.Printf("OCR识别结果: '%s'，未找到目标文字 '%s'\n", ocrResponse.Data, test)
			continue
		} else {
			// 识别失败
			fmt.Printf("OCR识别失败，错误代码: %d，错误信息: %s\n", ocrResponse.Code, ocrResponse.Data)
			continue
		}
	}

	// 检查输出中是否包含指定的应用名称
	return errors.New("找不到指定文本")
}

// ExtractTextFromArea
// @author: [Fantasia](https://www.npc0.com)
// @function: ExtractTextFromArea
// @description: 从指定区域提取所有文本（不验证特定内容）
// @param: x1, y1, x2, y2 int 左上角、右下角坐标
// @return: string, error
func ExtractTextFromArea(x1, y1, x2, y2 int) (string, error) {
	var imagePath string
	var err error

	// 提取图片
	if imagePath, err = ScreenshotGrayscale(x1, y1, x2, y2); err != nil {
		return "", err
	}
	// 移除图片
	defer func(name string) {
		_ = os.Remove(name)
	}(imagePath)

	// 读取图片
	var imageData []byte
	imageData, err = os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("读取图片文件失败: %v", err)
	}

	// 将图片转换为Base64编码
	base64Data := base64.StdEncoding.EncodeToString(imageData)

	// 构造请求参数
	requestData := map[string]interface{}{
		"Base64": base64Data,
	}

	// 将请求数据转换为JSON
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", fmt.Errorf("JSON编码失败: %v", err)
	}

	// 发送POST请求到PaddleOCR服务
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post("http://127.0.0.1:1224/api/ocr",
		"application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取并解析响应
	jsonData, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	var ocrResponse global.OCRResponse
	err = json.Unmarshal(jsonData, &ocrResponse)
	if err != nil {
		return "", fmt.Errorf("解析响应JSON失败: %v", err)
	}

	// 处理响应结果
	if ocrResponse.Code == 100 || ocrResponse.Code == 200 {
		return strings.TrimSpace(ocrResponse.Data), nil
	} else {
		return "", fmt.Errorf("OCR识别失败，错误代码: %d，错误信息: %s", ocrResponse.Code, ocrResponse.Data)
	}
}
