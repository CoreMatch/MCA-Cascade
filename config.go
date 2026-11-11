package mcacascade

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// DefaultQueryTimeout 是查询超时的默认值，与 VelocityGate 对齐：
// 分配查询发生在玩家落大厅之后，慢查询只延迟转向、不阻塞连接，
// 但也不能让玩家在大厅无谓等待。
const DefaultQueryTimeout = 2500 * time.Millisecond

// Config 是代理侧的路由配置。由宿主代理（Cascade）从自己的配置文件
// 装载后传入；本库不读任何文件。
//
//	Targets: 归一化 "host:port"（host 小写）→ 代理后端 ID。
//	MCA 成局携带的 join_target 在表里查出应送往的后端；映射放在
//	配置而非代码里，增删后端服无需重新构建。
type Config struct {
	BaseURL string
	Token   string
	Timeout time.Duration
	Targets map[string]string
}

// Validate 校验配置完整性。宿主代理应在启用路由前调用，
// 配错必须显式失败（拒绝启动），绝不静默降级。
func (c Config) Validate() error {
	if strings.TrimSpace(c.BaseURL) == "" {
		return errors.New("mca: base_url 必填")
	}
	u, err := url.Parse(c.BaseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("mca: base_url 必须是合法的 http(s) URL: %q", c.BaseURL)
	}
	if strings.TrimSpace(c.Token) == "" {
		return errors.New("mca: token 必填（OPS 服务令牌）")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("mca: query_timeout 必须为正数，当前 %v", c.Timeout)
	}
	for key, backend := range c.Targets {
		if NormalizeTarget(key) != key || key == "" {
			return fmt.Errorf("mca: targets 键 %q 不是归一化 host:port（host 小写、端口 1-65535）", key)
		}
		if strings.TrimSpace(backend) == "" {
			return fmt.Errorf("mca: targets[%s] 的后端 ID 为空", key)
		}
	}
	return nil
}
