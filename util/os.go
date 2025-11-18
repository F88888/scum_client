package util

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
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
// @function: 截屏取图片
// @description: 截取指定窗口句柄的图像（彩色图片，不再转换为灰度）
// @param: hand syscall.Handle 窗口句柄, x1, y1, x2, y2 int 裁剪区域坐标(可选，传0表示整个窗口)
// @param: enhance bool 是否启用图像增强（用于提高OCR识别准确率）
// @return: string, error
func ScreenshotGrayscale(hand syscall.Handle, x1, y1, x2, y2 int, enhance ...bool) (string, error) {
	// 生成文件地址，使用系统临时目录
	var f *os.File
	tempDir := os.TempDir()
	filePath := filepath.Join(tempDir, uuid.New().String()+".png")

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

	// 保存彩色图像（captureWindowImage返回的本来就是RGBA彩色图片）
	if f, err = os.Create(filePath); err != nil {
		return "", errors.New("创建图片文件失败:" + err.Error())
	}
	defer f.Close()

	// 直接保存RGBA图片（已经是彩色格式）
	rgbaImg, ok := finalImg.(*image.RGBA)
	if !ok {
		// 如果不是RGBA格式，转换为RGBA（通常不会发生，因为captureWindowImage返回的就是RGBA）
		bounds := finalImg.Bounds()
		rgbaImg = image.NewRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				rgbaImg.Set(x, y, finalImg.At(x, y))
			}
		}
	}

	// 如果启用增强，对图像进行增强处理
	shouldEnhance := false
	if len(enhance) > 0 && enhance[0] {
		shouldEnhance = true
	}

	if shouldEnhance {
		// 使用2.5倍放大，平衡识别准确率和处理速度
		rgbaImg = enhanceImageForOCR(rgbaImg, 2.5)
	}

	if err = png.Encode(f, rgbaImg); err != nil {
		return "", errors.New("保存图片失败:" + err.Error())
	}

	// 返回文件路径
	return filePath, nil
}

// enhanceImageForOCR 增强图像以提高OCR识别准确率
// @description: 对图像进行放大、锐化和对比度增强，提高小且模糊文字的识别准确率
// @param: img *image.RGBA 原始图像
// @param: scaleFactor float64 放大倍数（建议2.0-3.0）
// @return: *image.RGBA 增强后的图像
func enhanceImageForOCR(img *image.RGBA, scaleFactor float64) *image.RGBA {
	// 1. 放大图像
	enhanced := resizeImage(img, scaleFactor)

	// 2. 锐化处理
	enhanced = sharpenImage(enhanced)

	// 3. 对比度增强
	enhanced = enhanceContrast(enhanced)

	return enhanced
}

// resizeImage 使用双三次插值放大图像
// @description: 将图像放大指定倍数，使用双三次插值保持图像质量
// @param: img *image.RGBA 原始图像
// @param: scale float64 放大倍数
// @return: *image.RGBA 放大后的图像
func resizeImage(img *image.RGBA, scale float64) *image.RGBA {
	bounds := img.Bounds()
	oldWidth := bounds.Dx()
	oldHeight := bounds.Dy()
	newWidth := int(float64(oldWidth) * scale)
	newHeight := int(float64(oldHeight) * scale)

	resized := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// 双三次插值
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			// 计算原始图像中的对应位置
			srcX := float64(x) / scale
			srcY := float64(y) / scale

			// 获取周围4x4像素进行双三次插值
			var r, g, b, a float64
			var weightSum float64

			for dy := -1; dy <= 2; dy++ {
				for dx := -1; dx <= 2; dx++ {
					px := int(srcX) + dx
					py := int(srcY) + dy

					// 边界检查
					if px < 0 {
						px = 0
					}
					if px >= oldWidth {
						px = oldWidth - 1
					}
					if py < 0 {
						py = 0
					}
					if py >= oldHeight {
						py = oldHeight - 1
					}

					// 计算权重（双三次插值核函数）
					wx := cubicWeight(srcX - float64(px))
					wy := cubicWeight(srcY - float64(py))
					weight := wx * wy

					c := img.RGBAAt(px, py)
					r += float64(c.R) * weight
					g += float64(c.G) * weight
					b += float64(c.B) * weight
					a += float64(c.A) * weight
					weightSum += weight
				}
			}

			// 归一化
			if weightSum > 0 {
				r /= weightSum
				g /= weightSum
				b /= weightSum
				a /= weightSum
			}

			resized.SetRGBA(x, y, color.RGBA{
				R: uint8(math.Max(0, math.Min(255, r))),
				G: uint8(math.Max(0, math.Min(255, g))),
				B: uint8(math.Max(0, math.Min(255, b))),
				A: uint8(math.Max(0, math.Min(255, a))),
			})
		}
	}

	return resized
}

// cubicWeight 双三次插值权重函数
// @description: 计算双三次插值的权重
// @param: t float64 距离
// @return: float64 权重
func cubicWeight(t float64) float64 {
	t = math.Abs(t)
	if t <= 1 {
		return 1.5*t*t*t - 2.5*t*t + 1
	} else if t <= 2 {
		return -0.5*t*t*t + 2.5*t*t - 4*t + 2
	}
	return 0
}

// sharpenImage 锐化图像
// @description: 使用拉普拉斯算子对图像进行锐化处理
// @param: img *image.RGBA 原始图像
// @return: *image.RGBA 锐化后的图像
func sharpenImage(img *image.RGBA) *image.RGBA {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	sharpened := image.NewRGBA(bounds)

	// 拉普拉斯锐化核
	// [ 0 -1  0]
	// [-1  5 -1]
	// [ 0 -1  0]

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			// 获取周围像素
			c00 := img.RGBAAt(x-1, y-1)
			c01 := img.RGBAAt(x, y-1)
			c02 := img.RGBAAt(x+1, y-1)
			c10 := img.RGBAAt(x-1, y)
			c11 := img.RGBAAt(x, y) // 中心像素
			c12 := img.RGBAAt(x+1, y)
			c20 := img.RGBAAt(x-1, y+1)
			c21 := img.RGBAAt(x, y+1)
			c22 := img.RGBAAt(x+1, y+1)

			// 应用锐化核
			r := float64(c11.R)*5 - float64(c01.R+c10.R+c12.R+c21.R)
			g := float64(c11.G)*5 - float64(c01.G+c10.G+c12.G+c21.G)
			b := float64(c11.B)*5 - float64(c01.B+c10.B+c12.B+c21.B)
			a := c11.A

			// 限制范围
			r = math.Max(0, math.Min(255, r))
			g = math.Max(0, math.Min(255, g))
			b = math.Max(0, math.Min(255, b))

			sharpened.SetRGBA(x, y, color.RGBA{
				R: uint8(r),
				G: uint8(g),
				B: uint8(b),
				A: a,
			})
		}
	}

	// 复制边界像素
	for y := 0; y < height; y++ {
		sharpened.SetRGBA(0, y, img.RGBAAt(0, y))
		sharpened.SetRGBA(width-1, y, img.RGBAAt(width-1, y))
	}
	for x := 0; x < width; x++ {
		sharpened.SetRGBA(x, 0, img.RGBAAt(x, 0))
		sharpened.SetRGBA(x, height-1, img.RGBAAt(x, height-1))
	}

	return sharpened
}

// enhanceContrast 增强对比度
// @description: 使用对比度拉伸增强图像对比度
// @param: img *image.RGBA 原始图像
// @return: *image.RGBA 增强后的图像
func enhanceContrast(img *image.RGBA) *image.RGBA {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	enhanced := image.NewRGBA(bounds)

	// 计算最小和最大亮度值
	minBrightness := 255.0
	maxBrightness := 0.0

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := img.RGBAAt(x, y)
			brightness := 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
			if brightness < minBrightness {
				minBrightness = brightness
			}
			if brightness > maxBrightness {
				maxBrightness = brightness
			}
		}
	}

	// 如果对比度已经很高，不进行增强
	if maxBrightness-minBrightness < 50 {
		return img
	}

	// 对比度拉伸
	range_ := maxBrightness - minBrightness
	if range_ == 0 {
		return img
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := img.RGBAAt(x, y)
			brightness := 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)

			// 归一化到0-1范围
			normalized := (brightness - minBrightness) / range_

			// 拉伸到0-255范围，并应用轻微的S曲线增强
			enhancedBrightness := normalized * 255
			enhancedBrightness = math.Pow(enhancedBrightness/255, 0.9) * 255 // S曲线

			// 计算增强因子
			factor := enhancedBrightness / brightness
			if brightness == 0 {
				factor = 1.0
			}

			// 应用增强
			r := float64(c.R) * factor
			g := float64(c.G) * factor
			b := float64(c.B) * factor

			// 限制范围
			r = math.Max(0, math.Min(255, r))
			g = math.Max(0, math.Min(255, g))
			b = math.Max(0, math.Min(255, b))

			enhanced.SetRGBA(x, y, color.RGBA{
				R: uint8(r),
				G: uint8(g),
				B: uint8(b),
				A: c.A,
			})
		}
	}

	return enhanced
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
