// mcprobe —— MCA-Cascade 对局路由的最小 RW 客户端探针.
//
// 对 Cascade 游戏口做真实协议会话 (帧布局与 Avalon cascade_e2e 假客户端
// 一致: [4B 长度][4B 类型][负载], 字符串为 2 字节长度 + UTF-8), 验证
// "身份绑定后按 MCA 成局分配转向" 的两个可观测面:
//
//  1. 有分配 -> 收到 PACKET_RECONNECT_TO(178), 目标为代理跳转口, 真连落地:
//     mcprobe assigned <cascade游戏口> <玩家ID> <期望178目标host:port>
//  2. 无分配/查询失败 -> 窗口内没有 178, 玩家留在大厅:
//     mcprobe stay <cascade游戏口> <玩家ID> <等待秒数>
//     (期间收到的系统消息会打印出来, 便于验证 no_room_available 的可读原因)
//
// 探针在注册后会被引擎反作弊踢出, 这与 rwprobe 相同: 转向判定由 Cascade
// 在转发前完成, 抢在踢出之前收到 178 即为通过.
package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"time"
)

const (
	typeRegisterPlayer = 110
	typeChatBroadcast  = 141
	typePreregisterReq = 160
	typePreregisterAck = 161
	typeReconnectTo    = 178
)

func javaString(buf *[]byte, s string) {
	b := []byte(s)
	var l [2]byte
	binary.BigEndian.PutUint16(l[:], uint16(len(b)))
	*buf = append(*buf, l[:]...)
	*buf = append(*buf, b...)
}

func beInt(buf *[]byte, v int32) {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], uint32(v))
	*buf = append(*buf, b[:]...)
}

func frame(typeInt int, body []byte) []byte {
	f := make([]byte, 8+len(body))
	binary.BigEndian.PutUint32(f[0:4], uint32(len(body)))
	binary.BigEndian.PutUint32(f[4:8], uint32(typeInt))
	copy(f[8:], body)
	return f
}

func readFrame(conn net.Conn) (int, []byte, error) {
	var head [8]byte
	if _, err := io.ReadFull(conn, head[:]); err != nil {
		return 0, nil, err
	}
	length := binary.BigEndian.Uint32(head[0:4])
	typeInt := binary.BigEndian.Uint32(head[4:8])
	body := make([]byte, length)
	if _, err := io.ReadFull(conn, body); err != nil {
		return 0, nil, err
	}
	return int(typeInt), body, nil
}

func preregisterBody(roomCode string) []byte {
	var buf []byte
	javaString(&buf, "com.corrodinggames.rts")
	beInt(&buf, 2)
	beInt(&buf, 176)
	beInt(&buf, 0)
	buf = append(buf, 1) // query = true
	javaString(&buf, roomCode)
	return buf
}

// registerBody 与 Cascade protocol.DecodeRegister 的读取顺序对齐.
func registerBody(name, playerID string) []byte {
	var buf []byte
	javaString(&buf, "com.corrodinggames.rts")
	beInt(&buf, 2)
	beInt(&buf, 176)
	beInt(&buf, 0)
	javaString(&buf, name)
	buf = append(buf, 0) // 可空 string = false
	javaString(&buf, "mcprobe")
	javaString(&buf, playerID)
	return buf
}

// decodeChatMessage 解析 141 负载: string message + 其余字段此处不需要.
func decodeChatMessage(body []byte) string {
	if len(body) < 2 {
		return ""
	}
	n := int(binary.BigEndian.Uint16(body[0:2]))
	if 2+n > len(body) {
		return ""
	}
	return string(body[2 : 2+n])
}

// decodeReconnectTarget 解析 178 负载: byte / int32 / bool / int32 / string target.
func decodeReconnectTarget(body []byte) (string, error) {
	off := 1 + 4 + 1 + 4
	if off+2 > len(body) {
		return "", fmt.Errorf("178 负载缺目标串长度 (%d 字节)", len(body))
	}
	n := int(binary.BigEndian.Uint16(body[off : off+2]))
	off += 2
	if off+n > len(body) {
		return "", fmt.Errorf("178 目标串截断")
	}
	return string(body[off : off+n]), nil
}

// openSession 连接并完成 预注册(161) + 注册(110), 返回会话连接.
func openSession(addr, playerID string) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %v", err)
	}
	if _, err := conn.Write(frame(typePreregisterReq, preregisterBody(""))); err != nil {
		conn.Close()
		return nil, fmt.Errorf("写 Preregister 失败: %v", err)
	}
	if t, _, err := readFrame(conn); err != nil || t != typePreregisterAck {
		conn.Close()
		return nil, fmt.Errorf("预注册应答异常: t=%d err=%v", t, err)
	}
	fmt.Printf("  [1/3] 预注册 161 OK (默认路由 -> 大厅)\n")
	if _, err := conn.Write(frame(typeRegisterPlayer, registerBody("mcprobe", playerID))); err != nil {
		conn.Close()
		return nil, fmt.Errorf("写 Register 失败: %v", err)
	}
	fmt.Printf("  [2/3] 注册 110 已发 (玩家 %s), 等待 MCA 路由决策...\n", playerID)
	return conn, nil
}

func runAssigned(cascadeAddr, playerID, expectTarget string) error {
	conn, err := openSession(cascadeAddr, playerID)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
	var target string
	for {
		t, body, err := readFrame(conn)
		if err != nil {
			return fmt.Errorf("等待 178 时连接中断 (未收到跳转包): %v", err)
		}
		switch t {
		case typeReconnectTo:
			if target, err = decodeReconnectTarget(body); err != nil {
				return err
			}
		case typeChatBroadcast:
			if msg := decodeChatMessage(body); msg != "" {
				fmt.Printf("  [消息] %s\n", msg)
			}
			continue
		default:
			continue
		}
		break
	}
	fmt.Printf("  [3/3] 收到 PACKET_RECONNECT_TO(178), 目标 = %s\n", target)
	if target != expectTarget {
		return fmt.Errorf("178 目标 %q 与期望 %q 不符 (mca.targets/jump_port 配置检查)", target, expectTarget)
	}
	// 真连跳转口并预注册 = 落地目标后端.
	land, err := net.DialTimeout("tcp", target, 3*time.Second)
	if err != nil {
		return fmt.Errorf("落地连接失败: %v", err)
	}
	defer land.Close()
	_ = land.SetDeadline(time.Now().Add(10 * time.Second))
	if _, err := land.Write(frame(typePreregisterReq, preregisterBody(""))); err != nil {
		return fmt.Errorf("落地预注册写失败: %v", err)
	}
	if t, _, err := readFrame(land); err != nil || t != typePreregisterAck {
		return fmt.Errorf("落地预注册应答异常: t=%d err=%v", t, err)
	}
	fmt.Printf("  已落地 %s (预注册 161) —— 有分配真跳转全链路通\n", target)
	return nil
}

func runStay(cascadeAddr, playerID string, windowSec int) error {
	conn, err := openSession(cascadeAddr, playerID)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(time.Duration(windowSec) * time.Second))
	for {
		t, body, err := readFrame(conn)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				fmt.Printf("  [3/3] %ds 窗口内无 178 —— 玩家留在大厅 (符合预期)\n", windowSec)
				return nil
			}
			// 被引擎踢/断连也算"留在大厅": 没有发生转向.
			fmt.Printf("  [3/3] 连接结束 (%v), 期间无 178 —— 未发生转向 (符合预期)\n", err)
			return nil
		}
		switch t {
		case typeReconnectTo:
			target, _ := decodeReconnectTarget(body)
			return fmt.Errorf("窗口内收到 178 (目标 %s): 该玩家不应被转向", target)
		case typeChatBroadcast:
			if msg := decodeChatMessage(body); msg != "" {
				fmt.Printf("  [消息] %s\n", msg)
			}
		}
	}
}

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "用法:\n  mcprobe assigned <cascade游戏口> <玩家ID> <期望178目标>\n  mcprobe stay <cascade游戏口> <玩家ID> <等待秒数>")
		os.Exit(2)
	}
	mode, addr, playerID := os.Args[1], os.Args[2], os.Args[3]
	var err error
	switch mode {
	case "assigned":
		if len(os.Args) != 5 {
			fmt.Fprintln(os.Stderr, "assigned 模式需要 <期望178目标>")
			os.Exit(2)
		}
		err = runAssigned(addr, playerID, os.Args[4])
	case "stay":
		window := 5
		if len(os.Args) >= 5 {
			if window, err = strconv.Atoi(os.Args[4]); err != nil || window <= 0 {
				fmt.Fprintln(os.Stderr, "等待秒数必须是正整数")
				os.Exit(2)
			}
		}
		err = runStay(addr, playerID, window)
	default:
		fmt.Fprintf(os.Stderr, "未知模式 %q\n", mode)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		os.Exit(1)
	}
	fmt.Println("PASS")
}
