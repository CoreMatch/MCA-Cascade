# MCA-Cascade

MatchCore Arena 对局路由的 **Cascade 侧插件本体**：一个纯 Go 标准库依赖的
库模块，供 [Cascade](https://github.com/CoreMatch/Cascade)（Rusted
Warfare 边缘代理）在玩家身份绑定后查询 MCA 成局分配，并把已成局的玩家
送往对应后端子服。MC 侧的等价实现是
[MatchCoreArena-VelocityGate](https://github.com/CoreMatch/MatchCoreArena-VelocityGate)（Velocity
插件），两者的职责边界、失败兜底与配置形状保持对齐。

## 分层（不可推翻）

- **MCA 负责对局匹配**：输出分配结果（`assigned`、`join_target`、`reason`，
  含 `no_room_available` 语义）；
- **本库 + Cascade 扩展点负责把玩家送到指定子服**：查询
  `GET /api/matchmaking/assignment/{uuid}`（OPS 只读端点，契约
  `matchcorearena-business.yaml` 1.9.x/1.10.x），把 `join_target`
  （host:port）映射成 Cascade 后端 ID；
- **更细的"房间"层不属于这里**：`room` 字段有意不消费，具体房间由子服上的
  BedwarsRel-MCA 自行向 MCA 复核。

## 模块结构

| 文件 | 职责 |
|---|---|
| `assignment.go` | MCA 响应信封解析（`success`/`data`），失败显式报错，绝不伪造"无分配" |
| `client.go` | HTTP 客户端：Bearer OPS 令牌、超时、200/404/其余状态码三态语义 |
| `decide.go` | 纯决策：分配结果 + 目标映射 → `Outcome`（route / no_assignment / no_room / no_join_target / unknown_target） |
| `config.go` | 配置校验：必填项缺失显式失败 |

本库不依赖 Cascade、不读任何文件、不发起配置之外的 I/O；宿主代理负责：
装载配置、在正确的时机（玩家已认证且位于大厅）发起查询、执行跳转
（复用 `/goto` 的 `PACKET_RECONNECT_TO` + jump port 机制）、玩家消息与
后端健康/跳转口等运行时检查。

## 身份边界

分配端点按 HRPAuth 账号 UUID 解析（`users.uuid`，不查游戏绑定）。
因此只有已在 Cascade 完成 `/login` 或 `/claim` 绑定、拿到
`haUserUUID` 的玩家才可查询；游客没有可查询身份，永远留在大厅。
这与 VelocityGate 按 Minecraft UUID 查询同构。

## 失败语义（与 VelocityGate 对齐）

| 情况 | 行为 |
|---|---|
| MCA 超时/不可达/非 2xx/信封非法 | `ErrUnavailable`：留大厅，日志记为查询失败（不与无分配混淆） |
| 无有效分配 | 留大厅，静默 |
| `reason: no_room_available` | 留大厅，给玩家可读原因 |
| 成局未带 `join_target` | 留大厅 + 提示联系管理员 |
| `join_target` 未在映射表 | 留大厅 + 提示联系管理员 |
| 目标后端不健康/无跳转口（宿主侧检查） | 留大厅 + 提示稍后重试 |

## 与 Cascade 进程外插件系统的关系

Cascade 自 2026-11-12 起提供进程外插件系统（`cascade-plugin/v1`，
Cascade 仓 `docs/plugins/` 与 knowledge/0008）。本库的分配决策是**决策
产生方**：插件通过 `player.route` 事件（双模，授权后可否决）参与决策
边界——可否决一次转向（玩家留大厅并收到可读理由），或按观察档记录；
插件崩溃/超时一律 fail-open，不阻塞本库的路由链。本库本身保持为编译进
Cascade 的进程内组件（MCA HTTP 契约与映射决策），插件扩展点在宿主侧。

## 开发

```bash
go test ./...   # 全失败路径单测
```
