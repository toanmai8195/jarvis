#!/usr/bin/env bash
# execute-all watchdog: giữ execute-all chạy tiếp sau khi hết token / session chết.
#
#   .claude/scripts/execute-all-watchdog.sh start [UNTIL_PHASE]  # tạo tmux session `snaptix`: claude + watchdog + caffeinate
#   .claude/scripts/execute-all-watchdog.sh tick                 # kiểm tra một lần (watchdog gọi mỗi INTERVAL giây)
#   .claude/scripts/execute-all-watchdog.sh stop                 # dừng watchdog + caffeinate (để claude chạy tiếp)
#
# Mỗi tick: execute-all đang `running` mà
#   - claude không chạy trong window `claude`  → khởi động claude mới với "tiếp tục execute-all";
#   - claude chạy nhưng transcript không đổi IDLE_MIN phút (thường do hết token) → gõ "tiếp tục execute-all".
# Skill execute-all luôn `auto start` và suy trạng thái từ planning + git nên chạy lại an toàn;
# đang có subagent chạy thì skill chỉ kết thúc lượt, không spawn thêm.
# Chạy trong tmux (không dùng launchd) vì launchd không có quyền đọc ~/Documents.
set -u

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SESSION="${SESSION:-snaptix}"
INTERVAL="${INTERVAL:-1200}"
IDLE_MIN="${IDLE_MIN:-20}"
CLAUDE_CMD="${CLAUDE_CMD:-claude --chrome --permission-mode auto}"
LOG="$ROOT/.claude/state/watchdog.log"
TRANSCRIPTS="$HOME/.claude/projects/$(printf '%s' "$ROOT" | sed 's#[/.]#-#g')"
SELF="$ROOT/.claude/scripts/execute-all-watchdog.sh"

mkdir -p "$ROOT/.claude/state"
log() { echo "$(date '+%F %T') $*" | tee -a "$LOG"; }

auto_status() {
  python3 "$ROOT/.claude/scripts/planning.py" auto status 2>/dev/null |
    python3 -c 'import json,sys
try: print(json.load(sys.stdin).get("status", ""))
except Exception: print("")'
}

claude_running() {
  local pid
  pid=$(tmux list-panes -t "$SESSION:claude" -F '#{pane_pid}' 2>/dev/null | head -1)
  [ -n "$pid" ] && pgrep -P "$pid" >/dev/null
}

send() { tmux send-keys -t "$SESSION:claude" -l "$1" && sleep 1 && tmux send-keys -t "$SESSION:claude" Enter; }

cmd_tick() {
  local st
  st=$(auto_status)
  if [ "$st" != "running" ]; then
    log "execute-all: ${st:-chưa start} — không làm gì"
    return
  fi
  if ! tmux has-session -t "$SESSION" 2>/dev/null; then
    log "không có tmux session $SESSION — chạy '$SELF start' trước"
    return
  fi
  if ! claude_running; then
    log "claude không chạy → khởi động lại"
    send "cd '$ROOT' && $CLAUDE_CMD '/execute-all'"
    return
  fi
  if [ -n "$(find "$TRANSCRIPTS" -name '*.jsonl' -mmin "-$IDLE_MIN" 2>/dev/null | head -1)" ]; then
    log "đang chạy (transcript có thay đổi trong $IDLE_MIN phút)"
    return
  fi
  log "không tiến triển $IDLE_MIN phút → gõ 'tiếp tục execute-all'"
  send "tiếp tục execute-all"
}

cmd_start() {
  local until="${1:-8}"
  if tmux has-session -t "$SESSION" 2>/dev/null; then
    echo "tmux session $SESSION đã có: tmux attach -t $SESSION"
    exit 1
  fi
  tmux new-session -d -s "$SESSION" -n claude -c "$ROOT"
  send "$CLAUDE_CMD '/execute-all $until'"
  tmux new-window -d -t "$SESSION" -n watchdog -c "$ROOT" \
    "while true; do sleep $INTERVAL; bash '$SELF' tick; done"
  tmux new-window -d -t "$SESSION" -n caffeinate "caffeinate -dimsu"
  log "start: session $SESSION, execute-all đến phase $until, tick mỗi ${INTERVAL}s"
  echo "Xem: tmux attach -t $SESSION  (thoát xem: Ctrl-b d · đổi window: Ctrl-b 0/1/2)"
}

cmd_stop() {
  tmux kill-window -t "$SESSION:watchdog" 2>/dev/null
  tmux kill-window -t "$SESSION:caffeinate" 2>/dev/null
  log "stop: đã dừng watchdog + caffeinate"
}

case "${1:-tick}" in
  start) shift; cmd_start "$@" ;;
  tick) cmd_tick ;;
  stop) cmd_stop ;;
  *) sed -n '2,15p' "$0" ;;
esac
