#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
FRONTEND_DIR="$ROOT_DIR/frontend"
BACKEND_ENV="$BACKEND_DIR/.env"
BACKEND_BIN="${TMPDIR:-/tmp}/picflow-dev-$$"
BACKEND_PID=""
FRONTEND_PID=""
STOPPING=0

cleanup() {
  if [[ "$STOPPING" -eq 1 ]]; then
    return
  fi
  STOPPING=1
  trap - EXIT INT TERM

  echo
  echo "正在停止 PicFlow 本地开发服务..."
  [[ -n "$FRONTEND_PID" ]] && kill "$FRONTEND_PID" 2>/dev/null || true
  [[ -n "$BACKEND_PID" ]] && kill "$BACKEND_PID" 2>/dev/null || true
  [[ -n "$FRONTEND_PID" ]] && wait "$FRONTEND_PID" 2>/dev/null || true
  [[ -n "$BACKEND_PID" ]] && wait "$BACKEND_PID" 2>/dev/null || true
  rm -f "$BACKEND_BIN"
  echo "前后端已停止。"
}

trap 'cleanup; exit 130' INT TERM
trap cleanup EXIT

for command_name in go node npm; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "缺少命令：$command_name，请先安装后再运行。" >&2
    exit 1
  fi
done

if [[ ! -f "$BACKEND_ENV" ]]; then
  cp "$BACKEND_DIR/.env.example" "$BACKEND_ENV"
  echo "已创建 backend/.env"
fi

BACKEND_PORT="${PORT:-$(awk -F= '/^PORT=/{print $2; exit}' "$BACKEND_ENV")}"
BACKEND_PORT="${BACKEND_PORT:-8080}"
FRONTEND_PORT="${VITE_PORT:-5173}"

check_port() {
  local port="$1"
  local service_name="$2"
  if command -v lsof >/dev/null 2>&1 && lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "$service_name 端口 $port 已被占用。" >&2
    echo "如果是旧 Docker 容器，请先在项目根目录执行：docker compose stop" >&2
    exit 1
  fi
}

check_port "$BACKEND_PORT" "后端"
check_port "$FRONTEND_PORT" "前端"

if [[ ! -d "$FRONTEND_DIR/node_modules" ]]; then
  echo "首次运行，正在安装前端依赖..."
  (cd "$FRONTEND_DIR" && npm ci)
fi

echo "正在构建后端开发程序..."
(cd "$BACKEND_DIR" && GOCACHE="${GOCACHE:-/tmp/picflow-go-cache}" go build -o "$BACKEND_BIN" .)

echo "正在启动后端：http://localhost:$BACKEND_PORT"
cd "$BACKEND_DIR"
"$BACKEND_BIN" serve &
BACKEND_PID=$!
cd "$ROOT_DIR"

echo "正在启动前端：http://localhost:$FRONTEND_PORT"
"$FRONTEND_DIR/node_modules/.bin/vite" "$FRONTEND_DIR" --host 0.0.0.0 --port "$FRONTEND_PORT" --strictPort &
FRONTEND_PID=$!

echo
echo "PicFlow 本地开发环境已启动。"
echo "浏览器访问：http://localhost:$FRONTEND_PORT"
echo "后端健康检查：http://localhost:$BACKEND_PORT/health/ready"
echo "按 Ctrl+C 同时停止前后端。"

while true; do
  if ! kill -0 "$BACKEND_PID" 2>/dev/null; then
    wait "$BACKEND_PID" || true
    echo "后端进程已退出。" >&2
    exit 1
  fi
  if ! kill -0 "$FRONTEND_PID" 2>/dev/null; then
    wait "$FRONTEND_PID" || true
    echo "前端进程已退出。" >&2
    exit 1
  fi
  sleep 1
done
