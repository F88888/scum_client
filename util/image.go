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
	"qq_client/model/request"
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
}

// GetTextPositionFromCache 从缓存获取文本位置
// @description: 从缓存获取文本位置
// @param: text string 目标文本
// @return: *TextPositionCache, bool
func GetTextPositionFromCache(text string) (*TextPositionCache, bool) {
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
}

// getMultilingualTexts 获取多语言文本列表
// @description: 根据文本key获取所有支持的语言版本
// @param: textKey string 文本key（如 "MUTE", "GLOBAL" 等）
// @return: []string 多语言文本列表
func getMultilingualTexts(textKey string) []string {
	// 检查是否有多语言映射
	if texts, exists := global.GameUIText[strings.ToUpper(textKey)]; exists {
		return texts
	}
	// 如果没有映射，返回原始文本
	return []string{textKey}
}

// searchTextInFullScreen 全屏搜索文本并返回位置（支持多语言）
// @description: 全屏搜索文本并返回位置，支持多语言匹配
// @param: hand syscall.Handle 窗口句柄
// @param: targetText string 目标文本key（如 "MUTE", "GLOBAL" 等）
// @return: *TextPositionCache, error
func searchTextInFullScreen(hand syscall.Handle, targetText string) (*TextPositionCache, error) {
	// 获取多语言文本列表
	var ocrResult request.OcrResult
	textVariants := getMultilingualTexts(targetText)

	// 全屏截图
	imagePath, err := ScreenshotGrayscale(hand, 0, 0, global.GameWindowWidth, global.GameWindowHeight)
	if err != nil {
		return nil, fmt.Errorf("全屏截图失败: %v", err)
	}

	// 读取图片
	defer os.Remove(imagePath)
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("读取图片文件失败: %v", err)
	}

	// 将图片转换为Base64编码
	base64Data := base64.StdEncoding.EncodeToString(imageData)

	// 将请求数据转换为JSON
	jsonData, err := json.Marshal(map[string]interface{}{
		"image": base64Data,
	})

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
	if err = json.Unmarshal(responseData, &ocrResult); err != nil {
		return nil, fmt.Errorf("解析响应JSON失败: %v", err)
	}

	// 检查识别结果
	if ocrResult.Code == 101 {
		// 检测不到文本
		return nil, fmt.Errorf("OCR识别失败，code: %d (图片中无文本)", ocrResult.Code)
	}
	if ocrResult.Code != 100 {
		return nil, fmt.Errorf("OCR识别失败，code: %d", ocrResult.Code)
	}

	// 优先使用新的 items 数组格式，如果没有则回退到旧的 data 数组格式
	var itemsToProcess []request.OcrItem

	if len(ocrResult.Items) > 0 {
		// 使用新格式：items 数组
		itemsToProcess = ocrResult.Items
	} else {
		// 向后兼容：尝试解析 data 为数组
		dataArray, ok := ocrResult.Data.([]interface{})
		if ok {
			// 将旧格式转换为新格式
			for _, item := range dataArray {
				itemMap, ok := item.(map[string]interface{})
				if !ok {
					continue
				}

				text, _ := itemMap["text"].(string)
				confidence, _ := itemMap["confidence"].(float64)
				boxData, _ := itemMap["box"].([]interface{})

				// 转换 box 坐标
				var box [][]float64
				for _, point := range boxData {
					pointArray, ok := point.([]interface{})
					if !ok || len(pointArray) < 2 {
						continue
					}
					x, _ := pointArray[0].(float64)
					y, _ := pointArray[1].(float64)
					box = append(box, []float64{x, y})
				}

				// 提取 position（如果有）
				var position request.OcrPosition
				if posMap, ok := itemMap["position"].(map[string]interface{}); ok {
					if left, ok := posMap["left"].(float64); ok {
						position.Left = int(left)
					}
					if top, ok := posMap["top"].(float64); ok {
						position.Top = int(top)
					}
					if right, ok := posMap["right"].(float64); ok {
						position.Right = int(right)
					}
					if bottom, ok := posMap["bottom"].(float64); ok {
						position.Bottom = int(bottom)
					}
				}

				itemsToProcess = append(itemsToProcess, request.OcrItem{
					Text:       text,
					Confidence: confidence,
					Box:        box,
					Position:   position,
				})
			}
		}
	}

	if len(itemsToProcess) == 0 {
		return nil, fmt.Errorf("OCR响应中未找到识别结果")
	}

	// 遍历所有识别到的文本，查找目标文本（支持多语言）
	for _, item := range itemsToProcess {
		text := item.Text
		textUpper := strings.ToUpper(strings.TrimSpace(text))

		// 检查是否匹配任何语言版本
		matched := false
		for _, variant := range textVariants {
			variantUpper := strings.ToUpper(strings.TrimSpace(variant))
			if strings.Contains(textUpper, variantUpper) || strings.Contains(variantUpper, textUpper) {
				matched = true
				break
			}
		}

		if matched {
			// 找到目标文本，使用 position 或 box 计算坐标
			var x1, y1, x2, y2 int

			// 优先使用 position 字段（更简单直接）
			// 检查 position 是否有效（right 和 bottom 应该大于 left 和 top）
			if item.Position.Right > item.Position.Left && item.Position.Bottom > item.Position.Top {
				x1 = item.Position.Left
				y1 = item.Position.Top
				x2 = item.Position.Right
				y2 = item.Position.Bottom
			} else if len(item.Box) >= 4 {
				// 回退到使用 box 计算最小最大坐标
				var minX, minY, maxX, maxY float64
				minX, minY = 999999, 999999
				maxX, maxY = 0, 0

				for _, point := range item.Box {
					if len(point) < 2 {
						continue
					}
					x := point[0]
					y := point[1]

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

				x1 = int(minX)
				y1 = int(minY)
				x2 = int(maxX)
				y2 = int(maxY)
			} else {
				continue
			}

			// 创建缓存对象
			cache := &TextPositionCache{
				X1:    x1,
				Y1:    y1,
				X2:    x2,
				Y2:    y2,
				Found: true,
			}

			fmt.Printf("全屏搜索成功: 找到文本 '%s' (识别为: '%s', 置信度: %.2f) 在位置 [%d,%d,%d,%d]\n",
				targetText, text, item.Confidence, cache.X1, cache.Y1, cache.X2, cache.Y2)
			return cache, nil
		}
	}

	return nil, fmt.Errorf("全屏搜索未找到文本: '%s' (已尝试: %v)", targetText, textVariants)
}

// ExtractTextFromSpecifiedAreaAndValidateThreeTimes
// @author: [Fantasia](https://www.npc0.com)
// @function: ExtractTextFromSpecifiedAreaAndValidateThreeTimes
// @description: 从指定区域提取文本并验证三次（自动缓存位置，支持多语言）
// @param: hand syscall.Handle 窗口句柄
// @param: test string 期望识别的文本key（如 "MUTE", "GLOBAL" 等）
// @return: error
func ExtractTextFromSpecifiedAreaAndValidateThreeTimes(hand syscall.Handle, test string) error {
	// 检查缓存（使用原始key）
	cache, exists := GetTextPositionFromCache(test)

	var isNewlyFound bool
	if !exists {
		// 首次搜索，使用全屏搜索
		newCache, err := searchTextInFullScreen(hand, test)
		if err != nil {
			return err
		}

		// 保存到缓存
		setTextPositionCache(test, newCache)
		cache = newCache
		isNewlyFound = true
	} else {
		isNewlyFound = false
	}

	// 如果全屏搜索刚找到文本，直接返回成功（全屏搜索已经验证了文本存在）
	if isNewlyFound {
		return nil
	}

	// 使用缓存的位置进行识别（仅在使用已有缓存时验证）
	var ocrVerified bool
	var hasSuccessfulScreenshot bool
	for i := 1; i <= 3; i++ {
		// 截图指定区域
		imagePath, err := ScreenshotGrayscale(hand, cache.X1, cache.Y1, cache.X2, cache.Y2)
		if err != nil {
			fmt.Printf("[ERROR] 第%d次截图失败: %v\n", i, err)
			continue
		}

		// 获取图片文件信息
		_, err = os.Stat(imagePath)
		defer func(imgPath string) {
			// 删除临时文件
			_ = os.Remove(imgPath)
		}(imagePath)
		hasSuccessfulScreenshot = true

		// 读取图片
		imageData, err := os.ReadFile(imagePath)
		if err != nil {
			continue
		}

		// 将图片转换为Base64编码
		base64Data := base64.StdEncoding.EncodeToString(imageData)

		// 将请求数据转换为JSON
		jsonData, err := json.Marshal(map[string]interface{}{
			"image": base64Data,
		})
		if err != nil {
			continue
		}

		// 发送POST请求到PaddleOCR服务
		var ocrResult request.OcrResult
		client := &http.Client{Timeout: _const.OCRServiceAPITimeout}
		ocrAPIURL := fmt.Sprintf("http://%s:%d/api/ocr", global.OCRServiceHost, global.OCRServicePort)
		resp, err := client.Post(ocrAPIURL,
			"application/json", bytes.NewBuffer(jsonData))
		if err != nil {
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
		if err = json.Unmarshal(responseData, &ocrResult); err != nil {
			fmt.Printf("第%d次解析响应JSON失败: %v\n", i, err)
			continue
		}

		// 检查识别结果
		textVariants := getMultilingualTexts(test)
		if ocrResult.Code == 100 {
			// 识别成功，检查是否包含目标文本（支持多语言）
			// 优先使用新的 items 数组格式
			var itemsToProcess []request.OcrItem

			if len(ocrResult.Items) > 0 {
				// 使用新格式：items 数组
				itemsToProcess = ocrResult.Items
			} else {
				// 向后兼容：尝试解析 data 为数组
				dataArray, ok := ocrResult.Data.([]interface{})
				if ok {
					// 将旧格式转换为新格式
					for _, item := range dataArray {
						itemMap, ok := item.(map[string]interface{})
						if !ok {
							continue
						}

						text, _ := itemMap["text"].(string)
						confidence, _ := itemMap["confidence"].(float64)
						boxData, _ := itemMap["box"].([]interface{})

						// 转换 box 坐标
						var box [][]float64
						for _, point := range boxData {
							pointArray, ok := point.([]interface{})
							if !ok || len(pointArray) < 2 {
								continue
							}
							x, _ := pointArray[0].(float64)
							y, _ := pointArray[1].(float64)
							box = append(box, []float64{x, y})
						}

						// 提取 position（如果有）
						var position request.OcrPosition
						if posMap, ok := itemMap["position"].(map[string]interface{}); ok {
							if left, ok := posMap["left"].(float64); ok {
								position.Left = int(left)
							}
							if top, ok := posMap["top"].(float64); ok {
								position.Top = int(top)
							}
							if right, ok := posMap["right"].(float64); ok {
								position.Right = int(right)
							}
							if bottom, ok := posMap["bottom"].(float64); ok {
								position.Bottom = int(bottom)
							}
						}

						itemsToProcess = append(itemsToProcess, request.OcrItem{
							Text:       text,
							Confidence: confidence,
							Box:        box,
							Position:   position,
						})
					}
				}
			}

			if len(itemsToProcess) > 0 {
				for _, item := range itemsToProcess {
					text := item.Text
					textUpper := strings.ToUpper(strings.TrimSpace(text))

					// 检查是否匹配任何语言版本
					for _, variant := range textVariants {
						variantUpper := strings.ToUpper(strings.TrimSpace(variant))
						if strings.Contains(textUpper, variantUpper) || strings.Contains(variantUpper, textUpper) {
							ocrVerified = true
							return nil
						}
					}
				}
			}
			ocrVerified = true // OCR成功识别了文本，只是不匹配
		} else if ocrResult.Code == 200 {
			// Code 200: 可能表示"未找到目标文字"或"没有识别到文字"
			// 检查是否有识别结果
			var itemsToProcess []request.OcrItem
			if len(ocrResult.Items) > 0 {
				itemsToProcess = ocrResult.Items
			} else {
				// 尝试解析 data
				dataArray, ok := ocrResult.Data.([]interface{})
				if ok {
					for _, item := range dataArray {
						itemMap, ok := item.(map[string]interface{})
						if !ok {
							continue
						}
						text, _ := itemMap["text"].(string)
						confidence, _ := itemMap["confidence"].(float64)
						boxData, _ := itemMap["box"].([]interface{})
						var box [][]float64
						for _, point := range boxData {
							pointArray, ok := point.([]interface{})
							if !ok || len(pointArray) < 2 {
								continue
							}
							x, _ := pointArray[0].(float64)
							y, _ := pointArray[1].(float64)
							box = append(box, []float64{x, y})
						}
						var position request.OcrPosition
						if posMap, ok := itemMap["position"].(map[string]interface{}); ok {
							if left, ok := posMap["left"].(float64); ok {
								position.Left = int(left)
							}
							if top, ok := posMap["top"].(float64); ok {
								position.Top = int(top)
							}
							if right, ok := posMap["right"].(float64); ok {
								position.Right = int(right)
							}
							if bottom, ok := posMap["bottom"].(float64); ok {
								position.Bottom = int(bottom)
							}
						}
						itemsToProcess = append(itemsToProcess, request.OcrItem{
							Text:       text,
							Confidence: confidence,
							Box:        box,
							Position:   position,
						})
					}
				}
			}

			if len(itemsToProcess) > 0 {
				// 识别到了文字，但没有找到目标文字
				recognizedTexts := make([]string, 0, len(itemsToProcess))
				for _, item := range itemsToProcess {
					recognizedTexts = append(recognizedTexts, item.Text)
				}
			}
		}

		// 如果识别失败，等待后重试
		if i < 3 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// 判断失败原因
	if !hasSuccessfulScreenshot {
		// 如果所有截图都失败，保留缓存（位置可能是正确的，只是截图功能有问题）
		return errors.New("截图失败，无法验证文本")
	} else if ocrVerified {
		// 如果OCR成功识别了文本，但文本不匹配，说明位置可能已变化，清除缓存
		cacheMutex.Lock()
		delete(textPositionCache, test)
		cacheMutex.Unlock()
		return errors.New("文本位置已变化，缓存已清除")
	} else {
		// OCR识别失败，可能是临时问题，尝试全屏搜索一次确认文本是否还在
		newCache, err := searchTextInFullScreen(hand, test)
		if err == nil && newCache != nil {
			// 全屏搜索找到了文本，但位置已变化，更新缓存
			setTextPositionCache(test, newCache)
			return nil // 位置已更新，验证通过
		} else {
			// 全屏搜索也没找到，可能是界面已变化，保留旧缓存
			return errors.New("OCR识别失败，全屏搜索也未找到文本，保留缓存位置")
		}
	}
}

// ClickTextCenter
// @author: [Fantasia](https://www.npc0.com)
// @function: ClickTextCenter
// @description: 点击文本中心位置（使用窗口句柄）
// @param: hand syscall.Handle 窗口句柄
// @param: text string 目标文本
// @return: error
func ClickTextCenter(hand syscall.Handle, text string) error {
	// 获取文本位置（从缓存或全屏搜索）
	cache, exists := GetTextPositionFromCache(text)
	if !exists {
		// 首次搜索，使用全屏搜索
		newCache, err := searchTextInFullScreen(hand, text)
		if err != nil {
			return fmt.Errorf("全屏搜索文本 '%s' 失败: %v", text, err)
		}
		// 保存到缓存
		setTextPositionCache(text, newCache)
		cache = newCache
	} else {
		fmt.Printf("点击文本 '%s': 使用缓存位置 [%d,%d,%d,%d]\n", text, cache.X1, cache.Y1, cache.X2, cache.Y2)
	}

	// 计算中心坐标（窗口内坐标）
	centerX := (cache.X1 + cache.X2) / 2
	centerY := (cache.Y1 + cache.Y2) / 2
	fmt.Printf("点击文本 '%s': 计算中心坐标 (%d, %d) (文本区域: [%d,%d,%d,%d])\n",
		text, centerX, centerY, cache.X1, cache.Y1, cache.X2, cache.Y2)

	// 使用硬件级别的点击（适用于游戏窗口）
	fmt.Printf("点击文本 '%s': 正在通过硬件级别点击坐标 (%d, %d)...\n", text, centerX, centerY)
	if err := ClickWindowPosition(hand, centerX, centerY); err != nil {
		return fmt.Errorf("硬件级别点击失败: %v", err)
	}

	fmt.Printf("点击文本 '%s': 点击成功\n", text)
	return nil
}
