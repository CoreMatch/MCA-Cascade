#!/bin/bash
# run_scenario.sh —— MCA-Cascade 对局路由的栈上联调场景 (真实 Cascade +
# 真实 Avalon 后端 + docker redis + 桩 MCA).
#
# 前置 (由操作者完成, 脚本不代改共享栈):
#   1. /home/lnb/Desktop/avalon/stack 已起 (up.sh), 两后端健康;
#   2. docker redis 已起:  docker run -d --name mca-e2e-redis -p 127.0.0.1:6399:6379 redis:7
#   3. local/cascade/config.yaml 已临时加入:
#        redis_addr: "127.0.0.1:6399" / redis_prefix: "cascade:"
#        mca: enabled / base_url=http://127.0.0.1:9810 / token / targets
#        (target "127.0.0.1:5202" -> backend "sub-a")
#      并已用新配置重启 Cascade;
#   4. 玩家认证态已种入 redis (模拟 /login 成功后的状态):
#        redis-cli -p 6399 set "cascade:auth:e2e-player-a" "1111...1"
#        (b/c/d 同理: 2222.../3333.../4444...; 游客场景不种)
#
# 场景:
#   A 有分配     -> 178 目标 127.0.0.1:5199 且落地子服
#   B 无分配     -> 窗口内无 178, 留大厅
#   C 无可用房间 -> 无 178 + 可读原因消息
#   D 未登记目标 -> 无 178 + 联系管理员消息
#   E 游客       -> 无查询、无 178 (桩侧请求数不增)
#   F MCA 不可达 -> 停桩后有分配的玩家也留大厅 (查询失败兜底)
set -uo pipefail

HERE=$(cd "$(dirname "$0")" && pwd)
BIN=$HERE/bin/mcprobe
GAME=127.0.0.1:5123
JUMP_SUB=127.0.0.1:5199
STUB_PORT=9810
STUB_TOKEN=e2e-stub-token
fail=0

build_probe() {
  mkdir -p "$HERE/bin"
  (cd "$HERE/.." && go build -o "$HERE/bin/mcprobe" ./integration/mcprobe) || exit 1
}

check() { # $1=名字; 其余=命令
  local name=$1; shift
  echo "===== $name ====="
  if "$@"; then echo "PASS [$name]"; else echo "FAIL [$name]"; fail=1; fi
}

count_stub_requests() { # 桩日志里的请求行数
  grep -c '^\[stub-mca\]' "$HERE/stub_mca.log" 2>/dev/null || echo 0
}

build_probe

echo "===== 起桩 MCA (127.0.0.1:$STUB_PORT) ====="
pkill -f "stub_mca.py $STUB_PORT" 2>/dev/null; sleep 0.3
python3 "$HERE/stub_mca.py" "$STUB_PORT" "$STUB_TOKEN" >"$HERE/stub_mca.log" 2>&1 &
STUB_PID=$!
trap 'kill $STUB_PID 2>/dev/null' EXIT
sleep 0.5

check "A 有分配 -> 真跳转子服" "$BIN" assigned "$GAME" e2e-player-a "$JUMP_SUB"
check "B 无分配 -> 留大厅" "$BIN" stay "$GAME" e2e-player-b 4
check "C 无可用房间 -> 留大厅 + 可读原因" "$BIN" stay "$GAME" e2e-player-c 4
check "D 未登记目标 -> 留大厅 + 提示" "$BIN" stay "$GAME" e2e-player-d 4

echo "===== E 游客 (未绑定账号) -> 不得发起查询 ====="
before=$(count_stub_requests)
check "E 游客留大厅" "$BIN" stay "$GAME" e2e-player-guest 4
after=$(count_stub_requests)
if [ "$before" -eq "$after" ]; then
  echo "PASS [桩请求数未变 ($before) —— 游客未触发查询]"
else
  echo "FAIL [桩请求数 $before -> $after —— 游客不应触发查询]"
  fail=1
fi

echo "===== F MCA 不可达 -> 静默留大厅 (查询失败兜底) ====="
# 停桩 = MCA 整体不可达; 持有有效分配的玩家也不得被卡住/误跳转,
# Cascade 侧应记 WARN "assignment query failed" (见其日志).
kill $STUB_PID 2>/dev/null
sleep 0.5
check "F MCA 不可达, 有分配的玩家也留大厅" "$BIN" stay "$GAME" e2e-player-a 4
# 恢复桩 (供后续手工复跑); 退出时由 trap 统一清理
python3 "$HERE/stub_mca.py" "$STUB_PORT" "$STUB_TOKEN" >>"$HERE/stub_mca.log" 2>&1 &
STUB_PID=$!
sleep 0.5

echo
echo "===== 桩 MCA 请求记录 ====="
cat "$HERE/stub_mca.log"
echo
if [ "$fail" -eq 0 ]; then echo "==> 场景全部通过 (A-F)"; else echo "==> 存在失败场景"; exit 1; fi
