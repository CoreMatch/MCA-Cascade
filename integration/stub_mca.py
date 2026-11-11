#!/usr/bin/env python3
"""stub_mca —— MCA 分配端点的契约形状桩 (仅联调, 不是产品代码).

按 matchcorearena-business.yaml 的 MatchAssignment 信封应答
GET /api/matchmaking/assignment/{uuid}:

  {"success": true, "message": "...", "data": {...}}

场景表 (按 uuid 精确匹配, 均为 32 位无连字符形态, 对齐 HRPAuth users.uuid):

  1111...  assigned=true,  join_target=127.0.0.1:5202 (本机 stack 子服游戏口)
  2222...  assigned=false  (普通无分配, 静默留大厅)
  3333...  assigned=false, reason=no_room_available (留大厅 + 可读原因)
  4444...  assigned=true,  join_target=9.9.9.9:9 (未在映射表 -> 留大厅 + 提示)

用法:
  python3 stub_mca.py <端口> <OPS令牌>

鉴权: Authorization: Bearer <令牌>; 不符 -> 403 (同样走代理的查询失败兜底).
每个请求打印一行, 便于核对"谁被查了/查了几次".
"""

import json
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer

UUID_ASSIGNED = "1" * 32
UUID_NONE = "2" * 32
UUID_NO_ROOM = "3" * 32
UUID_UNKNOWN_TARGET = "4" * 32


def assignment_for(uuid: str):
    if uuid == UUID_ASSIGNED:
        return {
            "assigned": True,
            "group_id": 42,
            "match_type": "1v1",
            "formed_at": "2026-11-11T15:00:00Z",
            "team_id": "A",
            "room": "arena-1",
            "join_target": {"host": "127.0.0.1", "port": 5202},
        }
    if uuid == UUID_NO_ROOM:
        return {"assigned": False, "reason": "no_room_available"}
    if uuid == UUID_UNKNOWN_TARGET:
        return {
            "assigned": True,
            "group_id": 43,
            "match_type": "1v1",
            "team_id": "B",
            "join_target": {"host": "9.9.9.9", "port": 9},
        }
    return {"assigned": False}


class Handler(BaseHTTPRequestHandler):
    token = ""

    def log_message(self, fmt, *args):  # 静默 BaseHTTPRequestHandler 默认日志
        pass

    def do_GET(self):
        prefix = "/api/matchmaking/assignment/"
        if not self.path.startswith(prefix):
            self.send_response(404)
            self.end_headers()
            return
        uuid = self.path[len(prefix):]
        auth = self.headers.get("Authorization", "")
        if auth != f"Bearer {self.token}":
            print(f"[stub-mca] {uuid} -> 403 (令牌不符: {auth!r})", flush=True)
            self.send_response(403)
            self.end_headers()
            return
        data = assignment_for(uuid)
        body = json.dumps({"success": True, "message": "ok", "data": data}).encode()
        print(f"[stub-mca] {uuid} -> 200 assigned={data.get('assigned')}"
              f"{' reason=' + data['reason'] if data.get('reason') else ''}", flush=True)
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


def main():
    if len(sys.argv) != 3:
        print(__doc__, file=sys.stderr)
        sys.exit(2)
    port = int(sys.argv[1])
    Handler.token = sys.argv[2]
    server = HTTPServer(("127.0.0.1", port), Handler)
    print(f"[stub-mca] listening on 127.0.0.1:{port}", flush=True)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass


if __name__ == "__main__":
    main()
