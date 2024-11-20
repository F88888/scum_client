package server

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/atotto/clipboard"
	"github.com/go-vgo/robotgo"
	"io"
	"net/http"
	"qq_client/global"
	"qq_client/util"
	"regexp"
	"strings"
	"syscall"
	"time"
)

var flagsRegexp, _ = regexp.Compile("Page (\\d+)/(\\d+)")
var updateClipboard = map[string]bool{
	"#listflags 1 true":         true,
	"#ListSpawnedVehicles true": true,
	"#dumpallsquadsinfolist":    true,
}

// run
// @author: [Fantasia](https://www.npc0.com)
// @function: run
// @description: 获取运行命令
func run() string {
	// init
	var err error
	var bodyBytes []byte
	var req *http.Request
	var resp *http.Response
	httpClient := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}
	if req, err = http.NewRequest("GET", fmt.Sprintf(
		"%s/api/v1/run?id=%d", global.ScumConfig.ServerUrl, global.ScumConfig.ServerID), nil); err == nil {
		req.Header.Set("Content-Type", "application/json")
		if resp, err = httpClient.Do(req); err == nil {
			if bodyBytes, err = io.ReadAll(resp.Body); err == nil {
				return string(bodyBytes)
			}
		}
	}
	return ""
}

// squad
// @author: [Fantasia](https://www.npc0.com)
// @function: Send
// @description: 发送回调
func squad(body map[string]interface{}) {
	// init
	var err error
	var byteBody []byte
	var req *http.Request
	var resp *http.Response
	httpClient := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}
	if byteBody, err = json.Marshal(&body); err == nil {
		if req, err = http.NewRequest("POST", fmt.Sprintf(
			"%s/api/v1/squad", global.ScumConfig.ServerUrl), bytes.NewReader(byteBody)); err == nil {
			req.Header.Set("Content-Type", "application/json")
			if resp, err = httpClient.Do(req); err == nil {
				_, _ = io.ReadAll(resp.Body)
			}
		}
	}
}

// SaveChat
// @author: [Fantasia](https://www.npc0.com)
// @function: SaveChat
// @description: 回写聊天信息
func SaveChat(text string) {
	// init
	var err error
	var byteBody []byte
	var req *http.Request
	var resp *http.Response
	httpClient := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}
	if byteBody, err = json.Marshal(&text); err == nil {
		if req, err = http.NewRequest("POST", fmt.Sprintf(
			"%s/api/v1/recycling", global.ScumConfig.ServerUrl), bytes.NewReader(byteBody)); err == nil {
			req.Header.Set("Content-Type", "application/json")
			if resp, err = httpClient.Do(req); err == nil {
				_, _ = io.ReadAll(resp.Body)
			}
		}
	}
}

// Send
// @author: [Fantasia](https://www.npc0.com)
// @function: Send
// @description: 发送命令
func Send(hwnd syscall.Handle, text string) (out string, err error) {
	// init
	var allOk bool
	var read string
	util.SetForegroundWindow(hwnd)
	_ = clipboard.WriteAll("")
	robotgo.MoveClick(82, 319, "", false)
	_ = robotgo.KeyTap("a", "ctrl")
	if strings.Index(text, "502 Bad Gateway") != -1 {
		return out, nil
	}
	// 查询是否全局模式
	fmt.Println("----Send----", 1)
	if util.ExtractTextFromSpecifiedAreaAndValidateThreeTimes(233, 308, 267, 327, "GLOBAL") == nil {
		// 判断是否在游戏界面
		fmt.Println("----Send----", 2)
		if util.ExtractTextFromSpecifiedAreaAndValidateThreeTimes(30, 310, 61, 325, "MUTE") == nil {
			// 不在聊天界面
			_ = robotgo.KeyTap("a", "ctrl")
			// 延时2秒
			time.Sleep(1 * time.Second)
			if util.ExtractTextFromSpecifiedAreaAndValidateThreeTimes(
				30, 310, 61, 325, "MUTE") == nil {
				// 不在聊天界面, 退出聊天界面
				SaveChat(text)
				fmt.Println("----Send----", 3)
				return out, errors.New("reboot")
			}
		}
		// 判断是否本地模式
		fmt.Println("----Send----", 4)
		if util.ExtractTextFromSpecifiedAreaAndValidateThreeTimes(
			237, 309, 268, 328, "LOCAL") == nil {
			// 在管理员聊天界面
			fmt.Println("----Send----", 5)
			_ = robotgo.KeyTap("tab")
		} else if util.ExtractTextFromSpecifiedAreaAndValidateThreeTimes(
			233, 308, 267, 327, "ADMIN") == nil {
			// 在管理员聊天界面
			fmt.Println("----Send----", 6)
			_ = robotgo.KeyTap("tab")
			time.Sleep(1 * time.Second)
			_ = robotgo.KeyTap("tab")
		}
	}
	// 写入剪贴板
	var cache string
	fmt.Println("----Send----", 7, read)
	for i := 0; i < 8; i++ {
		// 查询剪贴板是否写入成功
		if regexpList := global.ExtractLocationRegexp.FindAllString(text, -1); len(regexpList) == 6 {
			cache = regexpList[0]
		} else {
			cache = text
		}
		_ = clipboard.WriteAll(cache)
		time.Sleep(time.Millisecond * 500)
		if read, err = clipboard.ReadAll(); err == nil && read != cache {
			_ = clipboard.WriteAll(cache)
		} else if err == nil && read == cache {
			break
		}
	}
	// 循环等待时间
	fmt.Println("----Send----", 8)
	time.Sleep(time.Millisecond * 100)
	for i := 0; i < 10; i++ {
		// 粘贴
		_ = robotgo.KeyTap("a", "ctrl")
		_ = robotgo.KeyTap("v", "ctrl")
		time.Sleep(time.Millisecond * 100)
		// 发送
		_ = robotgo.KeyTap("enter")
		fmt.Println("----Send----", 9)
		// 是否修改返回剪贴板
		if _, ok := updateClipboard[cache]; strings.Count(cache, "#listflags") != 0 || ok {
			// 延时
			fmt.Println("----Send----", 10)
			if out, err = clipboard.ReadAll(); err == nil && out != cache {
				fmt.Println("----Send----", 11)
				allOk = true
				break
			} else {
				fmt.Println("是否修改返回剪贴板", out)
				fmt.Println("----Send----", 12)
				if util.ExtractTextFromSpecifiedAreaAndValidateThreeTimes(
					233, 308, 267, 327, "GLOBAL") == nil {
					// 判断是否打开聊天界面
					robotgo.MoveClick(82, 319, "", false)
					_ = robotgo.KeyTap("a", "ctrl")
					time.Sleep(1 * time.Second)
					_ = robotgo.KeyTap("tab")
				}
				time.Sleep(time.Second * 1)
				if out, err = clipboard.ReadAll(); err == nil && out != cache {
					fmt.Println("----Send----", 13)
					allOk = true
					break
				}
				fmt.Println("执行输出", cache, out)
			}
			fmt.Println("----Send----", 14)
		} else {
			allOk = true
			fmt.Println("----Send----", 15)
			if cache == "#ListPlayers true" {
				time.Sleep(time.Second * 1)
				out, _ = clipboard.ReadAll()
			}
			break
		}
	}
	// 判断是否成功
	fmt.Println("----Send----", 16)
	if !allOk {
		return out, errors.New("reboot")
	}
	// 返回
	return out, nil
}

// ChatMonitor
// @author: [Fantasia](https://www.npc0.com)
// @function: ChatMonitor
// @description: 聊天监控信息
func ChatMonitor(hwnd syscall.Handle) {
	// init
	var i int
	var err error
	var out string
	_, _ = Send(hwnd, "#Teleport 0 0 0")
	for {
		// 延时
		time.Sleep(time.Millisecond * 150)
		if out = run(); out != "" {
			// 执行命令
			fmt.Println(out)
			if out, err = Send(hwnd, out); err != nil {
				if out, err = Send(hwnd, out); err != nil {
					return
				}
			}
		}
		// 被30整除
		if i%30 == 0 {
			// 获取载具列表
			if out, err = Send(hwnd, "#ListSpawnedVehicles true"); err != nil {
				return
			} else {
				squad(map[string]interface{}{
					"id":   global.ScumConfig.ServerID,
					"mode": "spawned",
					"info": out,
				})
			}
			// 获取用户列表
			if out, err = Send(hwnd, "#ListPlayers true"); err != nil {
				return
			} else {
				squad(map[string]interface{}{
					"id":   global.ScumConfig.ServerID,
					"mode": "user",
					"info": out,
				})
			}
		}
		if i%150 == 0 && i > 0 {
			// 被150整除
			var flagNum = 1
			for {
				// 获取队伍领地信息列表
				if out, err = Send(hwnd, fmt.Sprintf("#listflags %d true", flagNum)); err != nil {
					fmt.Println(fmt.Sprintf("#listflags %d true error:", flagNum), out, err.Error())
					return
				} else {
					// 发送领地信息
					squad(map[string]interface{}{
						"id":   global.ScumConfig.ServerID,
						"mode": "flags",
						"info": out,
					})
					// 判断是否退出
					if lenList := flagsRegexp.FindAllStringSubmatch(out, 1); len(lenList) > 0 {
						fmt.Println(lenList[0][1], lenList[0][2], lenList[0][1] == lenList[0][2])
						if lenList[0][1] == lenList[0][2] {
							fmt.Println("break")
							break
						}
					}
					// out length
					if len(out) < 10 {
						fmt.Println(out)
						break
					}
					// flagNum++
					flagNum += 1
				}
			}
			// 获取队伍用户列表
			if out, err = Send(hwnd, "#dumpallsquadsinfolist"); err != nil {
				fmt.Println("#dumpallsquadsinfolist error:", out, err.Error())
				return
			} else {
				squad(map[string]interface{}{
					"id":   global.ScumConfig.ServerID,
					"mode": "all_group",
					"info": out,
				})
			}
		}
		// i++
		i++
		if i > 1000 {
			i = 0
		}
	}
}
