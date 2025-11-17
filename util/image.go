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
	_const "qq_client/internal/const"
	"strings"
	"sync"
	"syscall"
	"time"
)

// TextPositionCache 文本位置缓存结构
type TextPositionCache struct {
	X1    int
	Y1    int
	X2    int
	Y2    int
	Found bool
}

// 全局文本位置缓存
var (
	textPositionCache = make(map[string]*TextPositionCache)
	cacheMutex        sync.RWMutex
)

// ClearTextPositionCache 清空文本位置缓存（进程重启时调用）
// @description: 清空文本位置缓存
func ClearTextPositionCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	textPositionCache = make(map[string]*TextPositionCache)
	fmt.Println("文本位置缓存已清空")
}

// getTextPositionFromCache 从缓存获取文本位置
// @description: 从缓存获取文本位置
// @param: text string 目标文本
// @return: *TextPositionCache, bool
func getTextPositionFromCache(text string) (*TextPositionCache, bool) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()
	cache, exists := textPositionCache[text]
	return cache, exists
}

// setTextPositionCache 设置文本位置缓存
// @description: 设置文本位置缓存
// @param: text string 目标文本
// @param: cache *TextPositionCache 缓存数据
func setTextPositionCache(text string, cache *TextPositionCache) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	textPositionCache[text] = cache
	fmt.Printf("缓存文本位置: '%s' -> [%d,%d,%d,%d]\n", text, cache.X1, cache.Y1, cache.X2, cache.Y2)
}

// searchTextInFullScreen 全屏搜索文本并返回位置
// @description: 全屏搜索文本并返回位置
// @param: hand syscall.Handle 窗口句柄
// @param: targetText string 目标文本
// @return: *TextPositionCache, error
func searchTextInFullScreen(hand syscall.Handle, targetText string) (*TextPositionCache, error) {
	fmt.Printf("开始全屏搜索文本: '%s'\n", targetText)

	// 全屏截图
	imagePath, err := ScreenshotGrayscale(hand, 0, 0, global.GameWindowWidth, global.GameWindowHeight)
	if err != nil {
		return nil, fmt.Errorf("全屏截图失败: %v", err)
	}
	defer os.Remove(imagePath)

	// 读取图片
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("读取图片文件失败: %v", err)
	}

	// 将图片转换为Base64编码
	base64Data := base64.StdEncoding.EncodeToString(imageData)

	// 将请求数据转换为JSON
	jsonData, err := json.Marshal(map[string]interface{}{
		"base64": base64Data,
		"options": map[string]interface{}{
			"data": map[string]interface{}{
				"format": "dict",
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("JSON编码失败: %v", err)
	}

	// 发送POST请求到PaddleOCR服务
	client := &http.Client{Timeout: _const.OCRServiceAPITimeout}
	ocrAPIURL := fmt.Sprintf("http://%s:%d/api/ocr", global.OCRServiceHost, global.OCRServicePort)
	resp, err := client.Post(ocrAPIURL,
		"application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取并解析响应
	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析OCR响应
	var ocrResult struct {
		Code int         `json:"code"`
		Data interface{} `json:"data"`
	}
	if err := json.Unmarshal(responseData, &ocrResult); err != nil {
		return nil, fmt.Errorf("解析响应JSON失败: %v", err)
	}

	// 检查识别结果
	if ocrResult.Code != 100 {
		return nil, fmt.Errorf("OCR识别失败，code: %d", ocrResult.Code)
	}

	// 解析data为数组
	dataArray, ok := ocrResult.Data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("OCR响应data格式错误")
	}

	// 遍历所有识别到的文本，查找目标文本
	for _, item := range dataArray {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		text, _ := itemMap["text"].(string)
		if strings.Contains(strings.ToUpper(text), strings.ToUpper(targetText)) {
			// 找到目标文本，提取box坐标
			boxData, ok := itemMap["box"].([]interface{})
			if !ok || len(boxData) < 4 {
				continue
			}

			// box格式: [[x1,y1], [x2,y2], [x3,y3], [x4,y4]] (顺时针四个角)
			// 我们需要找到最小和最大坐标
			var minX, minY, maxX, maxY float64
			minX, minY = 999999, 999999
			maxX, maxY = 0, 0

			for _, point := range boxData {
				pointArray, ok := point.([]interface{})
				if !ok || len(pointArray) < 2 {
					continue
				}
				x, _ := pointArray[0].(float64)
				y, _ := pointArray[1].(float64)

				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}

			// 创建缓存对象
			cache := &TextPositionCache{
				X1:    int(minX),
				Y1:    int(minY),
				X2:    int(maxX),
				Y2:    int(maxY),
				Found: true,
			}

			fmt.Printf("全屏搜索成功: 找到文本 '%s' 在位置 [%d,%d,%d,%d]\n",
				targetText, cache.X1, cache.Y1, cache.X2, cache.Y2)
			return cache, nil
		}
	}

	return nil, fmt.Errorf("全屏搜索未找到文本: '%s'", targetText)
}

// ExtractTextFromSpecifiedAreaAndValidateThreeTimes
// @author: [Fantasia](https://www.npc0.com)
// @function: ExtractTextFromSpecifiedAreaAndValidateThreeTimes
// @description: 从指定区域提取文本并验证三次（自动缓存位置）
// @param: hand syscall.Handle 窗口句柄
// @param: x1, y1, x2, y2 int 初始搜索区域（仅首次使用）
// @param: test string 期望识别的文本
// @return: error
func ExtractTextFromSpecifiedAreaAndValidateThreeTimes(hand syscall.Handle, test string) error {
	// 检查缓存
	cache, exists := getTextPositionFromCache(test)

	if !exists {
		// 首次搜索，使用全屏搜索
		fmt.Printf("首次搜索文本 '%s'，使用全屏搜索...\n", test)
		newCache, err := searchTextInFullScreen(hand, test)
		if err != nil {
			return err
		}

		// 保存到缓存
		setTextPositionCache(test, newCache)
		cache = newCache
	} else {
		fmt.Printf("使用缓存位置搜索文本 '%s': [%d,%d,%d,%d]\n",
			test, cache.X1, cache.Y1, cache.X2, cache.Y2)
	}

	// 使用缓存的位置进行识别
	for i := 1; i <= 3; i++ {
		// 截图指定区域
		imagePath, err := ScreenshotGrayscale(hand, cache.X1, cache.Y1, cache.X2, cache.Y2)
		if err != nil {
			fmt.Printf("第%d次截图失败: %v\n", i, err)
			continue
		}
		defer os.Remove(imagePath)

		// 读取图片
		imageData, err := os.ReadFile(imagePath)
		if err != nil {
			fmt.Printf("第%d次读取图片文件失败: %v\n", i, err)
			continue
		}

		// 将图片转换为Base64编码
		base64Data := base64.StdEncoding.EncodeToString(imageData)

		// 将请求数据转换为JSON
		jsonData, err := json.Marshal(map[string]interface{}{
			"base64": base64Data,
			"options": map[string]interface{}{
				"data": map[string]interface{}{
					"format": "dict",
				},
			},
		})
		if err != nil {
			fmt.Printf("第%d次JSON编码失败: %v\n", i, err)
			continue
		}

		// 发送POST请求到PaddleOCR服务
		client := &http.Client{Timeout: _const.OCRServiceAPITimeout}
		ocrAPIURL := fmt.Sprintf("http://%s:%d/api/ocr", global.OCRServiceHost, global.OCRServicePort)
		resp, err := client.Post(ocrAPIURL,
			"application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("第%d次发送请求失败: %v\n", i, err)
			continue
		}

		// 读取并解析响应
		responseData, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Printf("第%d次读取响应失败: %v\n", i, err)
			continue
		}

		// 解析OCR响应
		var ocrResult struct {
			Code int         `json:"code"`
			Data interface{} `json:"data"`
		}
		if err := json.Unmarshal(responseData, &ocrResult); err != nil {
			fmt.Printf("第%d次解析响应JSON失败: %v\n", i, err)
			continue
		}

		// 检查识别结果
		if ocrResult.Code == 100 {
			// 识别成功，检查是否包含目标文本
			dataArray, ok := ocrResult.Data.([]interface{})
			if ok {
				for _, item := range dataArray {
					itemMap, ok := item.(map[string]interface{})
					if !ok {
						continue
					}
					text, _ := itemMap["text"].(string)
					if strings.Contains(strings.ToUpper(text), strings.ToUpper(test)) {
						fmt.Printf("第%d次验证成功: 找到目标文字 '%s'\n", i, test)
						return nil
					}
				}
			}
		}

		// 如果识别失败，等待后重试
		if i < 3 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// 三次验证都失败，可能位置已变化，清除缓存
	fmt.Printf("验证失败: 未找到文本 '%s'，清除缓存\n", test)
	cacheMutex.Lock()
	delete(textPositionCache, test)
	cacheMutex.Unlock()

	return errors.New("找不到指定文本")
}

// ExtractTextFromArea
// @author: [Fantasia](https://www.npc0.com)
// @function: ExtractTextFromArea
// @description: 从指定区域提取所有文本（不验证特定内容）
// @param: x1, y1, x2, y2 int 左上角、右下角坐标
// @return: string, error
func ExtractTextFromArea(hand syscall.Handle, x1, y1, x2, y2 int) (string, error) {
	var imagePath string
	var err error

	// 提取图片
	if imagePath, err = ScreenshotGrayscale(hand, x1, y1, x2, y2); err != nil {
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
		"base64": base64Data,
		"options": map[string]interface{}{
			"data": map[string]interface{}{
				"format": "dict",
			},
		},
	}

	// 将请求数据转换为JSON
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", fmt.Errorf("JSON编码失败: %v", err)
	}

	// 发送POST请求到PaddleOCR服务
	client := &http.Client{Timeout: _const.OCRServiceAPITimeout}
	ocrAPIURL := fmt.Sprintf("http://%s:%d/api/ocr", global.OCRServiceHost, global.OCRServicePort)
	resp, err := client.Post(ocrAPIURL,
		"application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取并解析响应
	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析OCR响应
	var ocrResult struct {
		Code int         `json:"code"`
		Data interface{} `json:"data"`
	}
	if err := json.Unmarshal(responseData, &ocrResult); err != nil {
		return "", fmt.Errorf("解析响应JSON失败: %v", err)
	}

	// 处理响应结果
	if ocrResult.Code == 100 {
		// 识别成功，提取所有文本
		dataArray, ok := ocrResult.Data.([]interface{})
		if !ok {
			return "", fmt.Errorf("OCR响应data格式错误")
		}

		var texts []string
		for _, item := range dataArray {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			text, _ := itemMap["text"].(string)
			if text != "" {
				texts = append(texts, text)
			}
		}

		return strings.Join(texts, " "), nil
	} else if ocrResult.Code == 101 {
		// 图片中无文本
		return "", fmt.Errorf("图片中无文本")
	} else {
		// 识别失败
		dataStr, _ := ocrResult.Data.(string)
		return "", fmt.Errorf("OCR识别失败，错误代码: %d，错误信息: %s", ocrResult.Code, dataStr)
	}
}
