#!/usr/bin/env bash
set -euo pipefail

operation=$1
app=$2
executable="/opt/$app/$app"
state_dir="${XDG_STATE_HOME:-$HOME/.local/state}/$app"
launch_log="$state_dir/wsl-launch.log"
pid_file="$state_dir/wsl.pid"

fail() {
  printf '[WSL] %s\n' "$*" >&2
  exit 1
}

installed_pids() {
  local pid target
  for pid in $(pgrep -u "$(id -u)" || true); do
    target=$(readlink "/proc/$pid/exe" || true)
    if [[ "$target" == "$executable" || "$target" == "$executable (deleted)" ]]; then
      printf '%s\n' "$pid"
    fi
  done
}

case "$operation" in
  check)
    source /etc/os-release
    [[ "$ID" == ubuntu && "${VERSION_ID%%.*}" -ge 22 ]] || fail '目标需要 Ubuntu 22.04+'
    [[ $(dpkg --print-architecture) == amd64 ]] || fail '仅支持 Ubuntu x64'
    [[ $EUID -ne 0 ]] || fail 'WSL 默认用户不能是 root，请配置普通用户后重试'
    [[ -n "${WSL_INTEROP:-}" && -d /mnt/wslg ]] || fail '需要启用 WSL2 和 WSLg'
    [[ -n "${DISPLAY:-}" && -n "${WAYLAND_DISPLAY:-}" ]] || fail '缺少 WSLg 显示环境变量'
    printf '[WSL] %s，amd64，用户 %s\n' "$PRETTY_NAME" "$(id -un)"
    ;;
  graphics)
    # use_gfxredir=0 creates a taskbar entry but can leave its window transparent.
    attempts=20
    [[ "${3:-}" != wait ]] || attempts=120
    for ((i = 0; i < attempts; i++)); do
      mode=$(sed -n 's/.*use_gfxredir = //p' /mnt/wslg/weston.log 2>/dev/null | tail -1 || true)
      if [[ "$mode" == 1 && -S "${XDG_RUNTIME_DIR:-/mnt/wslg/runtime-dir}/$WAYLAND_DISPLAY" ]]; then
        exit 0
      fi
      sleep 0.25
    done
    if [[ "${mode:-}" == 0 ]]; then
      printf '[WSL] WSLg 共享内存重定向失败（COPY MODE）\n' >&2
      exit 42
    fi
    fail 'WSLg 图形会话未就绪，请确认 WSLg 已安装且支持 Linux GUI'
    ;;
  stop)
    pids=$(installed_pids)
    if [[ -n "$pids" ]]; then
      printf '[WSL] 关闭当前用户运行的已安装版 %s\n' "$app"
      while read -r pid; do kill -TERM "$pid"; done <<< "$pids"
      for ((i = 0; i < 120; i++)); do
        [[ -n $(installed_pids) ]] || exit 0
        sleep 0.25
      done
      fail '旧版本未退出，请关闭应用后重试'
    fi
    ;;
  install)
    deb=$3
    [[ $EUID -eq 0 ]] || fail '安装阶段需要 root'
    [[ $(dpkg-deb --field "$deb" Package) == "$app" ]] || fail '安装包名称与 mygo.json 的 slug 不一致'
    [[ $(dpkg-deb --field "$deb" Architecture) == amd64 ]] || fail '安装包必须为 amd64'
    export DEBIAN_FRONTEND=noninteractive
    if ! apt-get --simulate install "$deb" >/dev/null 2>&1; then
      apt-get update -o APT::Update::Error-Mode=any
    fi
    apt-get install -y --reinstall --no-install-recommends "$deb"
    [[ -f "$executable" ]] || fail "安装完成但未找到 $executable"
    # MyGo 0.9 packages Windows file modes when cross-building a .deb.
    # Restore the installed entry point's Linux executable mode before launching.
    chmod 0755 "$executable"
    printf '[WSL] 已安装 %s %s\n' "$app" "$(dpkg-query -W -f='${Version}' "$app")"
    ;;
  prepare)
    mkdir -p "$state_dir"
    : > "$launch_log"
    : > "$pid_file"
    ;;
  launch)
    exec > "$launch_log" 2>&1
    printf '%s\n' "$$" > "$pid_file"
    cd "$HOME"
    exec "$executable"
    ;;
  ready)
    for ((i = 0; i < 120; i++)); do
      pid=$(cat "$pid_file")
      if [[ "$pid" =~ ^[0-9]+$ && $(readlink "/proc/$pid/exe" || true) == "$executable" ]] &&
         grep -q '"msg":"editor_ready"' "$launch_log"; then
        printf '[WSL] 编辑器已加载，PID %s\n[WSL] 启动日志：%s\n' "$pid" "$launch_log"
        exit 0
      fi
      if [[ -n "$pid" ]] && ! kill -0 "$pid" 2>/dev/null; then
        break
      fi
      sleep 0.25
    done
    tail -30 "$launch_log" >&2
    fail "应用未完成启动，请查看 $launch_log"
    ;;
  *) fail "未知操作：$operation" ;;
esac
