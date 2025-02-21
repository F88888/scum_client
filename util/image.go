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
)

// ExtractTextFromSpecifiedAreaAndValidateThreeTimes
// @author: [Fantasia](https://www.npc0.com)
// @function: ExtractTextFromSpecifiedAreaAndValidateThreeTimes
// @description: 从指定区域提取文本并验证三次
// @param: x1, y1, x2, y2 int 左上角、右下角坐标
// @return: bool, error
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

		// 2. 将图片转换为Base64编码
		base64Data := base64.StdEncoding.EncodeToString(imageData)

		// 3. 构造请求参数
		requestData := global.OCRRequest{
			Base64: base64Data,
			Options: map[string]interface{}{
				"data.format": "text",
			},
		}

		// 将请求数据转换为JSON
		jsonData, err = json.Marshal(requestData)
		if err != nil {
			fmt.Printf("JSON编码失败: %v\n", err)
			continue
		}

		// 4. 发送POST请求
		var resp *http.Response
		resp, err = http.Post("http://127.0.0.1:1224/api/ocr",
			"application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("发送请求失败: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		// 5. 读取并解析响应
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

		// 6. 处理响应结果
		if ocrResponse.Code != 100 {
			fmt.Printf("OCR识别失败，错误代码: %d，错误信息: %s\n", ocrResponse.Code, ocrResponse.Data)
			continue
		}
		// 判断是否预期文本
		if strings.TrimSpace(ocrResponse.Data) == test {
			return nil
		}
	}
	// 检查输出中是否包含指定的应用名称
	return errors.New("找不到指定文本")
}
