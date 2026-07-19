#!/usr/bin/env bash
# start-codex-tmux.sh 用于在开发服务器上安装 tmux，并在当前项目根目录
# 启动或连接名为 awesome-zero-platform-codex 的 tmux session。
set -euo pipefail

SESSION_NAME="awesome-zero-platform-codex"
CODEX_CMD="env TERM=screen-256color COLORTERM=truecolor codex --dangerously-bypass-approvals-and-sandbox"
NO_ATTACH=0
TMUX_CONF="${HOME}/.tmux.conf"
TMUX_COLOR_BLOCK_BEGIN="# >>> start-codex-tmux color config >>>"
TMUX_COLOR_BLOCK_END="# <<< start-codex-tmux color config <<<"

err() {
  echo "Error: $*" >&2
}

info() {
  echo "==> $*"
}

need_sudo() {
  if [[ "${EUID}" -eq 0 ]]; then
    return 1
  fi
  return 0
}

run_as_root() {
  if need_sudo; then
    if ! command -v sudo >/dev/null 2>&1; then
      err "需要 root 权限安装 tmux，但当前用户不是 root，且系统中找不到 sudo。"
      exit 1
    fi
    sudo "$@"
  else
    "$@"
  fi
}

install_tmux() {
  info "tmux 未安装，开始自动安装。"

  if command -v apt-get >/dev/null 2>&1; then
    run_as_root apt-get update
    run_as_root apt-get install -y tmux
  elif command -v apt >/dev/null 2>&1; then
    run_as_root apt update
    run_as_root apt install -y tmux
  elif command -v dnf >/dev/null 2>&1; then
    run_as_root dnf install -y tmux
  elif command -v yum >/dev/null 2>&1; then
    run_as_root yum install -y tmux
  elif command -v apk >/dev/null 2>&1; then
    run_as_root apk add --no-cache tmux
  else
    err "找不到支持的包管理器，无法自动安装 tmux。请手动安装 tmux 后重试。"
    err "当前脚本支持的包管理器：apt/apt-get、dnf、yum、apk。"
    exit 1
  fi
}

ensure_tmux_config() {
  if [[ ! -f "${TMUX_CONF}" ]]; then
    touch "${TMUX_CONF}"
  fi

  if grep -Fqx "${TMUX_COLOR_BLOCK_BEGIN}" "${TMUX_CONF}"; then
    return 0
  fi

  {
    printf '\n%s\n' "${TMUX_COLOR_BLOCK_BEGIN}"
    printf '%s\n' 'set -g default-terminal "screen-256color"'
    printf '%s\n' "set -as terminal-overrides ',*:RGB'"
    printf '%s\n' 'set -g history-limit 100000'
    printf '%s\n' "${TMUX_COLOR_BLOCK_END}"
  } >>"${TMUX_CONF}"
}

usage() {
  cat <<EOF
Usage: $0 [--no-attach]

Options:
  --no-attach  只创建或确认 tmux session 存在，不 attach 到终端。
EOF
}

for arg in "$@"; do
  case "${arg}" in
    --no-attach)
      NO_ATTACH=1
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      err "未知参数：${arg}"
      usage >&2
      exit 1
      ;;
  esac
done

if ROOT_DIR=$(git rev-parse --show-toplevel 2>/dev/null); then
  :
else
  ROOT_DIR=$(pwd)
fi

if ! command -v tmux >/dev/null 2>&1; then
  install_tmux
fi

if ! command -v tmux >/dev/null 2>&1; then
  err "tmux 安装后仍不可用，请检查包管理器输出或手动安装 tmux。"
  exit 1
fi

ensure_tmux_config
export TERM=screen-256color
export COLORTERM=truecolor

if ! command -v codex >/dev/null 2>&1; then
  err "找不到 codex 命令。请先安装 Codex CLI，并确保 codex 在 PATH 中。"
  exit 1
fi

if ! tmux has-session -t "${SESSION_NAME}" 2>/dev/null; then
  info "创建 tmux session：${SESSION_NAME}"
  # 优先启用 Codex 所有权限：
  #   codex --dangerously-bypass-approvals-and-sandbox
  # 如果当前 Codex CLI 版本不支持该参数，可改为：
  #   codex --sandbox danger-full-access --approval-mode never
  tmux new-session -d -s "${SESSION_NAME}" -c "${ROOT_DIR}" "${CODEX_CMD}"
else
  info "tmux session 已存在，不重复创建或启动 codex：${SESSION_NAME}"
fi

if [[ "${NO_ATTACH}" -eq 1 ]]; then
  info "--no-attach 已启用，session 已准备好：${SESSION_NAME}"
  exit 0
fi

exec tmux attach-session -t "${SESSION_NAME}"
