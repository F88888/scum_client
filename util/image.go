package util

import (
	"os"
	"os/exec"
	"strings"
)

// AreaExtractTextSpecified
// @author: [Fantasia](https://www.npc0.com)
// @function: AreaExtractTextSpecified
// @description: 从指定区域提取文本
// @param: x1, y1, x2, y2 int 左上角、右下角坐标
// @return: bool, error
func AreaExtractTextSpecified(x1, y1, x2, y2 int) (string, error) {
	// init
	var err error
	var log []byte
	var imagePath string
	// 提取图片
	if imagePath, err = ScreenshotGrayscale(x1, y1, x2, y2); err == nil {
		defer os.Remove(imagePath)
		if log, err = exec.Command(
			"./Umi-OCR/Umi-OCR.exe", "--path", imagePath).Output(); err == nil {
			return strings.TrimSpace(string(log)), nil
		}
	}
	// 检查输出中是否包含指定的应用名称
	return "", err
}
