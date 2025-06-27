package rcon

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// RCON packet types
const (
	SERVERDATA_AUTH           = 3
	SERVERDATA_EXECCOMMAND    = 2
	SERVERDATA_RESPONSE_VALUE = 0
	SERVERDATA_AUTH_RESPONSE  = 2
)

// RCONClient represents an RCON client connection
type RCONClient struct {
	conn      net.Conn
	password  string
	requestID int32
}

// RCONPacket represents an RCON packet
type RCONPacket struct {
	Size  int32
	ID    int32
	Type  int32
	Body  string
	Empty byte
}

// NewRCONClient creates a new RCON client
func NewRCONClient(address, password string) (*RCONClient, error) {
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("连接RCON服务器失败: %v", err)
	}

	client := &RCONClient{
		conn:      conn,
		password:  password,
		requestID: 1,
	}

	// 进行身份验证
	if err := client.authenticate(); err != nil {
		conn.Close()
		return nil, err
	}

	return client, nil
}

// authenticate 身份验证
func (r *RCONClient) authenticate() error {
	packet := &RCONPacket{
		ID:   r.requestID,
		Type: SERVERDATA_AUTH,
		Body: r.password,
	}

	if err := r.sendPacket(packet); err != nil {
		return fmt.Errorf("发送认证包失败: %v", err)
	}

	response, err := r.readPacket()
	if err != nil {
		return fmt.Errorf("读取认证响应失败: %v", err)
	}

	if response.ID == -1 {
		return errors.New("RCON认证失败：密码错误")
	}

	r.requestID++
	return nil
}

// ExecuteCommand 执行命令
func (r *RCONClient) ExecuteCommand(command string) (string, error) {
	packet := &RCONPacket{
		ID:   r.requestID,
		Type: SERVERDATA_EXECCOMMAND,
		Body: command,
	}

	if err := r.sendPacket(packet); err != nil {
		return "", fmt.Errorf("发送命令失败: %v", err)
	}

	response, err := r.readPacket()
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	r.requestID++
	return response.Body, nil
}

// sendPacket 发送数据包
func (r *RCONClient) sendPacket(packet *RCONPacket) error {
	bodyLen := len(packet.Body)
	packet.Size = int32(bodyLen + 10) // 4 bytes ID + 4 bytes Type + body + 2 null bytes

	buffer := make([]byte, 0, packet.Size+4)

	// Size
	sizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBytes, uint32(packet.Size))
	buffer = append(buffer, sizeBytes...)

	// ID
	idBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(idBytes, uint32(packet.ID))
	buffer = append(buffer, idBytes...)

	// Type
	typeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(typeBytes, uint32(packet.Type))
	buffer = append(buffer, typeBytes...)

	// Body
	buffer = append(buffer, []byte(packet.Body)...)

	// Null terminators
	buffer = append(buffer, 0, 0)

	_, err := r.conn.Write(buffer)
	return err
}

// readPacket 读取数据包
func (r *RCONClient) readPacket() (*RCONPacket, error) {
	// 读取包大小
	sizeBuffer := make([]byte, 4)
	if _, err := r.conn.Read(sizeBuffer); err != nil {
		return nil, err
	}

	size := binary.LittleEndian.Uint32(sizeBuffer)

	// 读取包内容
	bodyBuffer := make([]byte, size)
	if _, err := r.conn.Read(bodyBuffer); err != nil {
		return nil, err
	}

	packet := &RCONPacket{
		Size: int32(size),
		ID:   int32(binary.LittleEndian.Uint32(bodyBuffer[0:4])),
		Type: int32(binary.LittleEndian.Uint32(bodyBuffer[4:8])),
	}

	// 提取Body（去除末尾的两个null字节）
	if size > 10 {
		packet.Body = string(bodyBuffer[8 : size-2])
	}

	return packet, nil
}

// Close 关闭连接
func (r *RCONClient) Close() error {
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

// QuickCommands 常用快速命令
type QuickCommands struct {
	client *RCONClient
}

// NewQuickCommands 创建快速命令执行器
func NewQuickCommands(client *RCONClient) *QuickCommands {
	return &QuickCommands{client: client}
}

// ListPlayers 列出玩家
func (qc *QuickCommands) ListPlayers() (string, error) {
	return qc.client.ExecuteCommand("#ListPlayers")
}

// ListVehicles 列出载具
func (qc *QuickCommands) ListVehicles() (string, error) {
	return qc.client.ExecuteCommand("#ListSpawnedVehicles")
}

// SetTime 设置时间
func (qc *QuickCommands) SetTime(hour, minute int) (string, error) {
	command := fmt.Sprintf("#SetTime %02d %02d", hour, minute)
	return qc.client.ExecuteCommand(command)
}

// TeleportPlayer 传送玩家
func (qc *QuickCommands) TeleportPlayer(playerName, targetPlayer string) (string, error) {
	command := fmt.Sprintf("#TeleportToPlayer %s %s", playerName, targetPlayer)
	return qc.client.ExecuteCommand(command)
}

// KickPlayer 踢出玩家
func (qc *QuickCommands) KickPlayer(playerName, reason string) (string, error) {
	command := fmt.Sprintf("#Kick %s %s", playerName, reason)
	return qc.client.ExecuteCommand(command)
}

// BanPlayer 封禁玩家
func (qc *QuickCommands) BanPlayer(playerName, reason string) (string, error) {
	command := fmt.Sprintf("#Ban %s %s", playerName, reason)
	return qc.client.ExecuteCommand(command)
}

// ExecuteBatch 批量执行命令
func (qc *QuickCommands) ExecuteBatch(commands []string) ([]string, error) {
	var results []string

	for _, cmd := range commands {
		result, err := qc.client.ExecuteCommand(cmd)
		if err != nil {
			results = append(results, fmt.Sprintf("错误: %v", err))
		} else {
			results = append(results, result)
		}
		// 短暂延迟避免命令冲突
		time.Sleep(50 * time.Millisecond)
	}

	return results, nil
}

// InteractiveMode 交互模式
func (qc *QuickCommands) InteractiveMode() {
	fmt.Println("=== SCUM RCON 快速命令工具 ===")
	fmt.Println("输入 'help' 查看帮助，输入 'exit' 退出")

	scanner := bufio.NewScanner()

	for {
		fmt.Print("SCUM> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if input == "exit" {
			break
		}

		if input == "help" {
			qc.printHelp()
			continue
		}

		// 处理预设命令别名
		command := qc.processAliases(input)

		result, err := qc.client.ExecuteCommand(command)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
		} else {
			fmt.Println(result)
		}
	}
}

// processAliases 处理命令别名
func (qc *QuickCommands) processAliases(input string) string {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return input
	}

	command := parts[0]
	args := parts[1:]

	aliases := map[string]string{
		"players":  "#ListPlayers",
		"vehicles": "#ListSpawnedVehicles",
		"time12":   "#SetTime 12 00",
		"time0":    "#SetTime 00 00",
		"flags":    "#listflags 1 true",
		"squads":   "#dumpallsquadsinfolist",
	}

	if alias, exists := aliases[command]; exists {
		if len(args) > 0 {
			return alias + " " + strings.Join(args, " ")
		}
		return alias
	}

	// 如果不是别名且不以#开头，自动添加#
	if !strings.HasPrefix(command, "#") {
		return "#" + input
	}

	return input
}

// printHelp 打印帮助信息
func (qc *QuickCommands) printHelp() {
	fmt.Println(`
快速命令别名:
  players          - 列出所有玩家 (#ListPlayers)
  vehicles         - 列出所有载具 (#ListSpawnedVehicles)
  time12           - 设置时间为中午12点 (#SetTime 12 00)
  time0            - 设置时间为午夜0点 (#SetTime 00 00)
  flags            - 列出所有标记 (#listflags 1 true)
  squads           - 列出所有小队 (#dumpallsquadsinfolist)

原始命令 (可省略#前缀):
  ListPlayers      - 列出玩家
  SetTime H M      - 设置时间 (H=小时, M=分钟)
  Kick 玩家名 原因  - 踢出玩家
  Ban 玩家名 原因   - 封禁玩家
  TeleportToPlayer 玩家1 玩家2 - 传送玩家

其他命令:
  help             - 显示此帮助
  exit             - 退出程序
`)
}
