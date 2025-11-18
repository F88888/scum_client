package util

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path"
	_const "qq_client/internal/const"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	gdi32DLL                   = syscall.NewLazyDLL("gdi32.dll")
	user32DLL_os               = syscall.NewLazyDLL("user32.dll")
	procCreateCompatibleDC     = gdi32DLL.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = gdi32DLL.NewProc("CreateCompatibleBitmap")
	procSelectObject           = gdi32DLL.NewProc("SelectObject")
	procBitBlt                 = gdi32DLL.NewProc("BitBlt")
	procDeleteObject           = gdi32DLL.NewProc("DeleteObject")
	procDeleteDC               = gdi32DLL.NewProc("DeleteDC")
	procGetDIBits              = gdi32DLL.NewProc("GetDIBits")
	procGetDC                  = user32DLL_os.NewProc("GetDC")
	procReleaseDC              = user32DLL_os.NewProc("ReleaseDC")
	procGetClientRect          = user32DLL_os.NewProc("GetClientRect")
	procPrintWindow            = user32DLL_os.NewProc("PrintWindow")
)

type RECT struct {
	Left, Top, Right, Bottom int32
}

type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type BITMAPINFO struct {
	BmiHeader BITMAPINFOHEADER
	BmiColors [1]uint32
}

// captureWindowImage 截取指定窗口的图像
// @description: 截取指定窗口的图像，支持最小化窗口截图，包含窗口状态检查和重试机制
func captureWindowImage(hwnd syscall.Handle) (*image.RGBA, error) {
	// 检查窗口句柄是否有效
	if hwnd == 0 {
		return nil, errors.New("窗口句柄无效")
	}

	if !IsWindow(hwnd) {
		return nil, errors.New("窗口句柄无效或窗口已关闭")
	}

	// 检查窗口是否最小化
	isMinimized := IsIconic(hwnd)
	if !isMinimized {
		// 窗口未最小化时，尝试激活窗口以获得更好的截图效果（但不强制）
		// 如果窗口不可见，尝试显示它
		if !IsWindowVisible(hwnd) {
			ShowWindow(hwnd, 5)
			time.Sleep(50 * time.Millisecond)
		}
		// 尝试将窗口置于前台（可选，不影响截图） - 已注释：使用句柄操作不需要窗口置顶
		// SetForegroundWindow(hwnd)
		// BringWindowToTop(hwnd)
		// 等待窗口内容渲染
		time.Sleep(100 * time.Millisecond)
	}

	var lastErr error
	for attempt := 1; attempt <= _const.ScreenshotMaxRetries; attempt++ {

		img, err := captureWindowImageInternal(hwnd, isMinimized)
		if err == nil {
			return img, nil
		}

		lastErr = err

		// 如果是窗口状态相关错误，重试
		if strings.Contains(err.Error(), "无法获取窗口") ||
			strings.Contains(err.Error(), "无法获取位图数据") ||
			strings.Contains(err.Error(), "无法复制窗口内容") {
			if attempt < _const.ScreenshotMaxRetries {
				time.Sleep(_const.ScreenshotRetryDelay)
				continue
			}
		} else {
			// 其他错误直接返回
			return nil, err
		}
	}

	return nil, fmt.Errorf("截图失败（重试%d次）: %v", _const.ScreenshotMaxRetries, lastErr)
}

// captureWindowImageInternal 截取指定窗口的图像（内部实现）
// @description: 实际的截图实现，不包含重试逻辑，优先使用 PrintWindow 以支持 DirectX/OpenGL 渲染的窗口和最小化窗口
// @param: hwnd syscall.Handle 窗口句柄
// @param: isMinimized bool 窗口是否最小化
func captureWindowImageInternal(hwnd syscall.Handle, isMinimized bool) (*image.RGBA, error) {
	// 获取窗口的客户区域大小
	var rect RECT
	const PW_RENDERFULLCONTENT = 0x00000002
	ret, _, _ := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		return nil, errors.New("无法获取窗口客户区域")
	}

	width := int(rect.Right - rect.Left)
	height := int(rect.Bottom - rect.Top)

	if width <= 0 || height <= 0 {
		return nil, errors.New("窗口大小无效")
	}

	// 获取屏幕设备上下文（用于创建兼容的DC）
	hdcScreen, _, _ := procGetDC.Call(0) // 0 表示屏幕DC
	if hdcScreen == 0 {
		return nil, errors.New("无法获取屏幕设备上下文")
	}
	defer procReleaseDC.Call(0, hdcScreen)

	// 创建兼容的设备上下文
	hdcMem, _, _ := procCreateCompatibleDC.Call(hdcScreen)
	if hdcMem == 0 {
		return nil, errors.New("无法创建兼容设备上下文")
	}
	defer procDeleteDC.Call(hdcMem)

	// 创建兼容的位图
	hBitmap, _, _ := procCreateCompatibleBitmap.Call(hdcScreen, uintptr(width), uintptr(height))
	if hBitmap == 0 {
		return nil, errors.New("无法创建兼容位图")
	}
	defer procDeleteObject.Call(hBitmap)

	// 选择位图到内存设备上下文
	procSelectObject.Call(hdcMem, hBitmap)

	// 优先使用 PrintWindow API，支持 DirectX/OpenGL 渲染的窗口和最小化窗口
	// PW_RENDERFULLCONTENT = 0x00000002 (Windows 8.1+)
	// 这个标志可以捕获使用硬件加速渲染的内容，并且支持最小化窗口
	if ret, _, _ = procPrintWindow.Call(uintptr(hwnd), hdcMem, PW_RENDERFULLCONTENT); ret == 0 {
		// PrintWindow 失败
		if isMinimized {
			// 最小化窗口只能使用 PrintWindow，如果失败则返回错误
			return nil, errors.New("无法使用 PrintWindow 捕获最小化窗口内容")
		}

		// 获取窗口的设备上下文
		hdcWindow, _, _ := procGetDC.Call(uintptr(hwnd))
		if hdcWindow == 0 {
			return nil, errors.New("无法获取窗口设备上下文")
		}
		defer procReleaseDC.Call(uintptr(hwnd), hdcWindow)

		// 将窗口内容复制到内存设备上下文
		const SRCCOPY = 0x00CC0020
		ret, _, _ = procBitBlt.Call(hdcMem, 0, 0, uintptr(width), uintptr(height), hdcWindow, 0, 0, SRCCOPY)
		if ret == 0 {
			return nil, errors.New("无法复制窗口内容（BitBlt 和 PrintWindow 都失败，可能是窗口使用硬件加速渲染）")
		}
	}

	// 准备位图信息结构
	bi := BITMAPINFO{
		BmiHeader: BITMAPINFOHEADER{
			BiSize:        uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
			BiWidth:       int32(width),
			BiHeight:      -int32(height), // 负值表示从上到下的位图
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: 0, // BI_RGB
		},
	}

	// 分配像素数据缓冲区
	pixelDataSize := width * height * 4 // RGBA 每像素4字节
	pixelData := make([]byte, pixelDataSize)

	// 获取位图数据
	ret, _, _ = procGetDIBits.Call(
		hdcMem,
		hBitmap,
		0,
		uintptr(height),
		uintptr(unsafe.Pointer(&pixelData[0])),
		uintptr(unsafe.Pointer(&bi)),
		0, // DIB_RGB_COLORS
	)
	if ret == 0 {
		return nil, errors.New("无法获取位图数据")
	}

	// 创建 RGBA 图像
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// 将 BGRA 格式转换为 RGBA 格式
	for i := 0; i < len(pixelData); i += 4 {
		b := pixelData[i]
		g := pixelData[i+1]
		r := pixelData[i+2]
		a := pixelData[i+3]

		// 计算在图像中的位置
		pixelIndex := i / 4
		x := pixelIndex % width
		y := pixelIndex / width

		img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: a})
	}

	return img, nil
}

// ScreenshotGrayscale
// @author: [Fantasia](https://www.npc0.com)
// @function: 截屏取灰度图片
// @description: 截取指定窗口句柄的图像并转换为灰度图
// @param: hand syscall.Handle 窗口句柄, x1, y1, x2, y2 int 裁剪区域坐标(可选，传0表示整个窗口)
// @return: string, error
func ScreenshotGrayscale(hand syscall.Handle, x1, y1, x2, y2 int) (string, error) {
	// 生成文件地址
	var f *os.File
	var filePath = path.Join("D:/", uuid.New().String()+".png")

	// 截取窗口图像
	img, err := captureWindowImage(hand)
	if err != nil {
		return "", errors.New("无法截取窗口图像:" + err.Error())
	}

	// 如果指定了裁剪区域，进行裁剪
	var finalImg image.Image = img
	if x2 > x1 && y2 > y1 {
		bounds := img.Bounds()
		// 确保裁剪区域在图像范围内
		if x1 < bounds.Min.X {
			x1 = bounds.Min.X
		}
		if y1 < bounds.Min.Y {
			y1 = bounds.Min.Y
		}
		if x2 > bounds.Max.X {
			x2 = bounds.Max.X
		}
		if y2 > bounds.Max.Y {
			y2 = bounds.Max.Y
		}

		// 创建裁剪后的图像
		cropRect := image.Rect(x1, y1, x2, y2)
		croppedImg := image.NewRGBA(cropRect)
		for y := y1; y < y2; y++ {
			for x := x1; x < x2; x++ {
				croppedImg.Set(x, y, img.At(x, y))
			}
		}
		finalImg = croppedImg
	}

	// 将图像转换为灰度图像
	grayImg := image.NewGray(finalImg.Bounds())
	for y := finalImg.Bounds().Min.Y; y < finalImg.Bounds().Max.Y; y++ {
		for x := finalImg.Bounds().Min.X; x < finalImg.Bounds().Max.X; x++ {
			pixel := finalImg.At(x, y)
			gray := color.GrayModel.Convert(pixel).(color.Gray)
			grayImg.Set(x, y, gray)
		}
	}

	// 保存灰度图像
	if f, err = os.Create(filePath); err != nil {
		return "", errors.New("创建图片文件失败:" + err.Error())
	}
	defer f.Close()

	if err = png.Encode(f, grayImg); err != nil {
		return "", errors.New("保存灰度图像失败:" + err.Error())
	}

	// 返回文件路径
	return filePath, nil
}

// SpecifiedCoordinateColor
// @author: [Fantasia](https://www.npc0.com)
// @function: 获取指定坐标颜色
// @description: 使用窗口句柄获取指定坐标颜色
// @param: hand syscall.Handle 窗口句柄, x1, y1 int 坐标
// @return: string 颜色值，格式为十六进制字符串(如"FF0000")
func SpecifiedCoordinateColor(hand syscall.Handle, x1, y1 int) string {
	// 截取指定窗口的图像
	img, err := captureWindowImage(hand)
	if err != nil {
		// 如果截取失败，返回默认颜色
		return "000000"
	}

	// 检查坐标是否在图像范围内
	bounds := img.Bounds()
	if x1 < bounds.Min.X || x1 >= bounds.Max.X || y1 < bounds.Min.Y || y1 >= bounds.Max.Y {
		// 如果坐标超出范围，返回默认颜色
		return "000000"
	}

	// 获取指定坐标的颜色
	pixelColor := img.At(x1, y1)
	// 将颜色转换为 RGBA 格式
	rgba := color.RGBAModel.Convert(pixelColor).(color.RGBA)

	// 将 RGBA 转换为十六进制字符串 (格式为 RRGGBB)
	return fmt.Sprintf("%02X%02X%02X", rgba.R, rgba.G, rgba.B)
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
