package util

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"
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
	var log []byte
	var imagePath string
	for {
		i += 1
		if i > 3 {
			break
		}
		// 提取图片
		if imagePath, err = ScreenshotGrayscale(x1, y1, x2, y2); err != nil {
			continue
		}
		// 提取文本
		if log, err = exec.Command("./Umi-OCR/Umi-OCR.exe", "--path", imagePath).Output(); err != nil {
			_ = os.Remove(imagePath)
			continue
		}
		// 判断是否预期文本
		_ = os.Remove(imagePath)
		if strings.TrimSpace(string(log)) == test {
			return nil
		}
		// 延时
		time.Sleep(time.Millisecond * 200)
	}
	// 检查输出中是否包含指定的应用名称
	return errors.New("找不到指定文本")
}
