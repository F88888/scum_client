package server

import (
	"fmt"
	"github.com/go-vgo/robotgo"
	"os/exec"
	"qq_client/util"
	"strings"
	"syscall"
	"time"
)

// 错误计算
var errorNumber int
var errorNumber2 int

// ErrorReboot
// @author: [Fantasia](https://www.npc0.com)
// @function: ErrorReboot
// @description: 错误重启
func ErrorReboot() {
	// 判断错误次数
	if errorNumber > 15 || errorNumber2 > 100 {
		// 错误次数大于10，重启游戏
		cmd := exec.Command("taskkill", "/IM", "SCUM.exe", "/F")
		_ = cmd.Run()
	}
}

// Start
// @author: [Fantasia](https://www.npc0.com)
// @function: 启动服务主逻辑
// @description: 机器人登录检测主逻辑
func Start() {
	// init
	var ok bool
	var err error
	ErrorReboot()
	var text string
	var hwnd syscall.Handle
	fmt.Println("重启机器人....")
	//hwnd = util.GetForegroundWindow()
	//fmt.Println(util.GetClassName(hwnd))
	//fmt.Println(util.GetWindowText(hwnd))
	// 判断是否有scum游戏进程
	if ok, err = util.CheckIfAppRunning("SCUM"); err != nil || !ok {
		// 启动游戏
		cmd := exec.Command("cmd", "/C", "start", "", "steam://rungameid/513710")
		fmt.Println("游戏未启动,启动游戏")
		_ = cmd.Start()
		errorNumber++
		// 延时30秒
		time.Sleep(30 * time.Second)
		return
	}
	// 查找窗口句柄
	if hwnd = util.FindWindow("UnrealWindow", "SCUM  "); hwnd == 0 {
		cmd := exec.Command("cmd", "/C", "start", "", "steam://rungameid/513710")
		fmt.Println("游戏窗口未找到!!!")
		_ = cmd.Start()
		// 延时120秒
		time.Sleep(120 * time.Second)
		errorNumber++
		return
	}
	// 设置游戏窗口大小
	util.MoveWindow(hwnd, 8, 31, 857, 593)
	// 设置游戏窗口置顶
	util.SetForegroundWindow(hwnd)
	// 判断是否在登录页面
	if text, err = util.AreaExtractTextSpecified(
		66, 395, 168, 421); err == nil && strings.Index(text, "CONTINUE") != -1 {
		// 在登录界面判断是否有机器人
		fmt.Println("目前在登录界面,判断是否有机器人....")
		robotgo.MoveClick(426, 348, "enter", false)
		if util.SpecifiedCoordinateColor(97, 142) != "ffffff" {
			// 没有机器人,切换机器人模式
			fmt.Println("没有机器人,切换机器人模式....")
			if err = robotgo.KeyTap("d", "ctrl"); err != nil {
				fmt.Println("切换机器人模式失败:", err)
				errorNumber++
				return
			}
			// 延时5秒
			time.Sleep(1 * time.Second)
			errorNumber++
			return
		}
		// 点击登录
		fmt.Println("登录中....")
		robotgo.MoveClick(97, 405, "enter", false)
	}
	// 判断是否在加载界面
	if util.SpecifiedCoordinateColor(427, 142) == "ffffff" && util.SpecifiedCoordinateColor(
		438, 153) == "ffffff" {
		// 延时120秒
		fmt.Println("加载界面....")
		time.Sleep(1 * time.Second)
		errorNumber2++
		return
	}
	// 判断是否在游戏界面
	if text, err = util.AreaExtractTextSpecified(
		30, 310, 61, 325); err == nil && strings.Index(text, "MUTE") != -1 {
		// 不在聊天界面
		fmt.Println("已进入游戏界面....")
		_ = robotgo.KeyTap("t")
		// 延时2秒
		time.Sleep(1 * time.Second)
		errorNumber2 = 0
		errorNumber = 0
	}
	// 判断是否本地模式
	if text, err = util.AreaExtractTextSpecified(
		237, 309, 268, 328); err == nil && strings.Index(text, "LOCAL") != -1 {
		// 在管理员聊天界面
		_ = robotgo.KeyTap("tab")
	} else if text, err = util.AreaExtractTextSpecified(
		233, 308, 267, 327); err == nil && strings.Index(text, "GLOBAL") != -1 {
		// 全局聊天界面, 启动机器人聊天模式
		fmt.Println("全局聊天...")
		ChatMonitor(hwnd)
	} else if text, err = util.AreaExtractTextSpecified(
		233, 308, 267, 327); err == nil && strings.Index(text, "ADMIN") != -1 {
		// 在管理员聊天界面
		_ = robotgo.KeyTap("tab")
		time.Sleep(1 * time.Second)
		_ = robotgo.KeyTap("tab")
	}
	// 延时5秒
	_ = robotgo.KeyTap("t")
	time.Sleep(1 * time.Second)
}
