package mcacascade

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Assignment 是 MCA 成局分配查询结果（GET /api/matchmaking/assignment/{uuid}
// 响应信封的 data 段，契约 matchcorearena-business.yaml 1.9.x/1.10.x）。
// 只保留代理路由所需字段；1.10.0 房间层的 room 字段在这里有意不消费——
// 代理只负责把玩家送到 join_target 指向的子服，具体房间由子服上的
// BedwarsRel-MCA 自行向 MCA 复核。
type Assignment struct {
	// Assigned 为 true 表示玩家当前持有有效成局分配。
	Assigned bool
	// Reason 仅在 Assigned 为 false 但存在有效成局时给出原因
	//（目前只有 no_room_available：预占过期/清单变更）；
	// 普通"无分配"为空。路由行为与无分配一致（留在大厅），
	// 但调用方应给玩家一句可读提示。
	Reason string
	// GroupID/MatchType/TeamID 仅用于日志可观测。
	GroupID   int64
	MatchType string
	TeamID    string
	// JoinHost/JoinPort 是成局绑定的子服地址（MCA 侧配置了就带，
	// 没配置则省略）。JoinHost 为空或 JoinPort <= 0 表示未携带。
	JoinHost string
	JoinPort int
}

// ReasonNoRoom 与 MCA 服务端常量一致（契约 1.10.0）：
// 有效成局但无法提供房间。
const ReasonNoRoom = "no_room_available"

// assignmentEnvelope 是 MCA 统一响应信封的最小投影。
type assignmentEnvelope struct {
	Success bool `json:"success"`
	Data    struct {
		Assigned   bool   `json:"assigned"`
		Reason     string `json:"reason"`
		GroupID    int64  `json:"group_id"`
		MatchType  string `json:"match_type"`
		TeamID     string `json:"team_id"`
		JoinTarget *struct {
			Host string `json:"host"`
			Port int    `json:"port"`
		} `json:"join_target"`
	} `json:"data"`
}

// HasJoinTarget 报告成局是否携带了可用的子服地址。
func (a *Assignment) HasJoinTarget() bool {
	return a != nil && a.JoinHost != "" && a.JoinPort > 0
}

// JoinTargetKey 返回归一化后的 "host:port" 键（host 小写）；
// 未携带地址时返回空串。
func (a *Assignment) JoinTargetKey() string {
	if !a.HasJoinTarget() {
		return ""
	}
	return NormalizeTarget(fmt.Sprintf("%s:%d", a.JoinHost, a.JoinPort))
}

// ParseAssignmentResponse 解析 MCA 响应体。信封 success != true 或缺少
// data 对象视为 MCA 不可用（返回错误），绝不降级成伪造的"无分配"——
// 路由侧对不可用与无分配的处置虽然同为留在大厅，但日志语义不同。
func ParseAssignmentResponse(body []byte) (*Assignment, error) {
	var env assignmentEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("解析 MCA 响应失败: %w", err)
	}
	if !env.Success {
		return nil, errors.New("MCA 响应信封 success != true")
	}
	asn := &Assignment{
		Assigned:  env.Data.Assigned,
		Reason:    env.Data.Reason,
		GroupID:   env.Data.GroupID,
		MatchType: env.Data.MatchType,
		TeamID:    env.Data.TeamID,
	}
	if env.Data.JoinTarget != nil {
		asn.JoinHost = env.Data.JoinTarget.Host
		asn.JoinPort = env.Data.JoinTarget.Port
	}
	return asn, nil
}
