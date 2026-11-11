package mcacascade

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrUnavailable 表示 MCA 查询不可用（不可达、超时、非预期状态码、
// 响应不可解析）。调用方必须把它与"无分配"区分记录：两者对玩家的
// 处置相同（留在大厅），但一个是故障、一个是业务常态。
var ErrUnavailable = errors.New("MCA 分配查询不可用")

// maxResponseBytes 限制响应体大小，防止异常上游撑爆内存。
// 分配响应正常只有几百字节。
const maxResponseBytes = 1 << 20

// maxUUIDLen 限制查询键长度。查询键是 HRPAuth 账号 UUID
// （users.uuid，32 位无连字符十六进制），放宽到 64 兼容形态演化；
// 空键直接拒绝。
const maxUUIDLen = 64

// Client 是 MCA 成局分配查询客户端。实现必须保证：查询失败返回
// ErrUnavailable（可 errors.Is 判定），绝不静默吞掉后返回伪造的
// "无分配"。
type Client interface {
	// Query 按玩家 UUID 查询当前有效成局分配。userUUID 必须是
	// 玩家已绑定账号的 HRPAuth 账号 UUID（游客没有可查询身份，
	// 调用方不应发起查询）。
	Query(ctx context.Context, userUUID string) (*Assignment, error)
}

// HTTPClient 是 Client 的 HTTP 实现：
// GET {base}/api/matchmaking/assignment/{uuid}，Bearer 使用 OPS 服务令牌。
//
// 状态码语义（对齐 VelocityGate HttpMcaClient）：
//   - 200 → 按信封解析（含 assigned=false 与 reason）；
//   - 404 → 视为无分配（1.9.1 起未知 UUID 已改答 200 assigned:false，
//     这里保留 404 容忍只是防御旧版本）；
//   - 其余非 2xx、超时与网络错误 → ErrUnavailable。
type HTTPClient struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewHTTPClient 构造 Client。baseURL 形如 https://arena.mcnb.dev（尾部
// 斜杠会被去掉）；token 为 MCA tokens.yaml 中的 OPS 服务令牌；
// timeout <= 0 时取默认 2500ms。
func NewHTTPClient(baseURL, token string, timeout time.Duration) *HTTPClient {
	if timeout <= 0 {
		timeout = 2500 * time.Millisecond
	}
	return &HTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http: &http.Client{
			Timeout: timeout,
		},
	}
}

// Query 实现 Client。
func (c *HTTPClient) Query(ctx context.Context, userUUID string) (*Assignment, error) {
	if userUUID == "" || len(userUUID) > maxUUIDLen {
		return nil, fmt.Errorf("%w: 非法玩家 UUID（空或过长），拒绝发起查询", ErrUnavailable)
	}
	endpoint := c.baseURL + "/api/matchmaking/assignment/" + url.PathEscape(userUUID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: 构造请求失败: %v", ErrUnavailable, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// 未知 UUID = 无分配（1.9.1 语义），不是故障。
		return &Assignment{}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: MCA 返回非预期状态码 %d", ErrUnavailable, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: 读取响应失败: %v", ErrUnavailable, err)
	}
	asn, err := ParseAssignmentResponse(body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return asn, nil
}
