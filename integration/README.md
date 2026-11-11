# integration —— 栈上联调工具

| 文件 | 作用 |
|---|---|
| `mcprobe/` | 最小 RW 客户端探针：真实协议会话，断言"有分配 → 178 真跳转落地 / 无分配 → 留大厅" |
| `stub_mca.py` | MCA 分配端点的契约形状桩（信封/字段对齐 matchcorearena-business.yaml，按 uuid 分场景） |
| `run_scenario.sh` | 场景编排：A 有分配 / B 无分配 / C 无可用房间 / D 未登记目标 / E 游客不查询 / F MCA 不可达 |

## 验证边界（如实）

- 真实：Cascade 二进制、Avalon 大厅/子服后端、跳转口监听、178 落地、
  玩家绑定表（SQLite）、redis 认证缓存读取路径；
- 桩替：MCA 服务（契约形状桩）、玩家认证态（直接种 redis，
  等价于 /login 成功后的 `SetAuthenticated` 结果）。
  本机栈不含 HRPAuth，`/login` 凭据核验腿不在本联调范围内。
