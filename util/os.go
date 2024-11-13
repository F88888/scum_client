package util

import (
	"bytes"
	"errors"
	"github.com/go-vgo/robotgo"
	"github.com/google/uuid"
	"github.com/vova616/screenshot"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path"
	"strings"
)

// ScreenshotGrayscale
// @author: [Fantasia](https://www.npc0.com)
// @function: 截屏取灰度图片
// @description: 检查指定的应用是否在Windows任务列表中运行
// @param: x1, y1, x2, y2 int 左上角、右下角坐标
// @return: string, error
func ScreenshotGrayscale(x1, y1, x2, y2 int) (string, error) {
	// 生成文件地址
	var f *os.File
	var filePath = path.Join(os.TempDir(), uuid.New().String()+".png")
	// 获取屏幕图像
	img, err := screenshot.CaptureRect(image.Rect(x1, y1, x2, y2))
	if err != nil {
		return "", errors.New("无法截取屏幕图像:" + err.Error())
	}
	// 将图像转换为灰度图像
	grayImg := image.NewGray(img.Bounds())
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			pixel := img.At(x, y)
			gray := color.GrayModel.Convert(pixel).(color.Gray)
			grayImg.Set(x, y, gray)
		}
	}
	// 可以保存灰度图像
	if f, err = os.Create(filePath); err != nil {
		return "", errors.New("创建图片文件失败:" + err.Error())
	}
	// 储存
	defer f.Close()
	if err = png.Encode(f, grayImg); err != nil {
		return "", errors.New("保存灰度图像失败:" + err.Error())
	}
	// 返回文件路径
	return filePath, err
}

// SpecifiedCoordinateColor
// @author: [Fantasia](https://www.npc0.com)
// @function: 获取指定坐标颜色
// @description: 获取指定坐标颜色
// @param: x1, y1 int 坐标
// @return: string
func SpecifiedCoordinateColor(x1, y1 int) string {
	// 获取屏幕图像
	return robotgo.GetPixelColor(x1, y1)
}

// CheckIfAppRunning
// @author: [Fantasia](https://www.npc0.com)
// @function: CheckIfAppRunning
// @description: 检查指定的应用是否在Windows任务列表中运行
// @param: appName 参数是你想要检查的应用的名称。
// @return: bool, error
func CheckIfAppRunning(appName string) (bool, error) {
	// 使用task list命令列出所有运行的应用
	cmd := exec.Command("tasklist")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return false, err
	}

	// 检查输出中是否包含指定的应用名称
	return strings.Contains(out.String(), appName), nil
}
