package mcacascade

import (
	"fmt"
	"strconv"
	"strings"
)

// Outcome 是一次分配查询后的路由决策结果。查询本身失败（ErrUnavailable）
// 不经过 Decide，由调用方自行记为查询失败——两者都让玩家留在大厅，
// 但日志语义不同。
type Outcome int

const (
	// OutcomeNoAssignment：无有效分配，留大厅，无需提示。
	OutcomeNoAssignment Outcome = iota
	// OutcomeNoRoom：有效成局但无可用房间（reason=no_room_available，
	// 预占过期/清单变更），留大厅，调用方应给玩家可读原因。
	OutcomeNoRoom
	// OutcomeRoute：把玩家送往 Decision.BackendID。
	OutcomeRoute
	// OutcomeNoJoinTarget：成局有效但未携带子服地址（MCA 侧未配置）。
	OutcomeNoJoinTarget
	// OutcomeUnknownTarget：join_target 未在映射表登记。
	OutcomeUnknownTarget
)

// String 供日志使用。
func (o Outcome) String() string {
	switch o {
	case OutcomeNoAssignment:
		return "no_assignment"
	case OutcomeNoRoom:
		return "no_room_available"
	case OutcomeRoute:
		return "routed"
	case OutcomeNoJoinTarget:
		return "no_join_target"
	case OutcomeUnknownTarget:
		return "unknown_target"
	default:
		return fmt.Sprintf("outcome(%d)", int(o))
	}
}

// Decision 是 Decide 的完整输出。
type Decision struct {
	Outcome   Outcome
	BackendID string // 仅 OutcomeRoute 有效
	// 以下为日志可观测字段。
	Reason     string // MCA 给出的原因（如 no_room_available）
	GroupID    int64
	MatchType  string
	TeamID     string
	JoinTarget string // 归一化后的 host:port（未携带时为空）
}

// Decide 依据配置的目标映射把一次成功的分配查询转换为路由决策。
// 纯函数，不发起任何 I/O；asn 为 nil 时按无分配处理（防御调用方）。
//
// 目标后端的健康、跳转口存在性等运行时检查属于代理侧职责
// （只有代理知道自己的后端目录），不在本库范围内。
func Decide(cfg Config, asn *Assignment) Decision {
	if asn == nil {
		return Decision{Outcome: OutcomeNoAssignment}
	}
	d := Decision{
		Reason:    asn.Reason,
		GroupID:   asn.GroupID,
		MatchType: asn.MatchType,
		TeamID:    asn.TeamID,
	}
	if !asn.Assigned {
		if asn.Reason != "" {
			d.Outcome = OutcomeNoRoom
		} else {
			d.Outcome = OutcomeNoAssignment
		}
		return d
	}
	key := asn.JoinTargetKey()
	d.JoinTarget = key
	if key == "" {
		d.Outcome = OutcomeNoJoinTarget
		return d
	}
	backendID, ok := cfg.Targets[key]
	if !ok || backendID == "" {
		d.Outcome = OutcomeUnknownTarget
		return d
	}
	d.Outcome = OutcomeRoute
	d.BackendID = backendID
	return d
}

// NormalizeTarget 归一化 "host:port"：去空白、host 转小写、端口必须为
// 1-65535 的纯数字；非法输入返回空串。映射表的键与查询键都必须经过
// 它，保证大小写/空白差异不影响命中。
func NormalizeTarget(raw string) string {
	t := strings.TrimSpace(raw)
	colon := strings.LastIndex(t, ":")
	if colon <= 0 || colon == len(t)-1 {
		return ""
	}
	host := strings.ToLower(strings.TrimSpace(t[:colon]))
	portStr := t[colon+1:]
	if host == "" || portStr == "" {
		return ""
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return ""
	}
	return host + ":" + strconv.Itoa(port)
}
