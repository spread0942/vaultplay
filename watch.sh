#!/usr/bin/env bash
# Vaultplay launcher: hardened rootless Podman + loopback socat bridge.
set -euo pipefail

IMAGE="${VAULTPLAY_IMAGE:-localhost/vaultplay:latest}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PORT_START="${VAULTPLAY_PORT_START:-8787}"
PORT_END="${VAULTPLAY_PORT_END:-8887}"

usage() {
  cat <<'EOF'
Usage:
  ./watch.sh /path/to/folder-or-file.mp4
  ./watch.sh --test-isolation
  ./watch.sh --help

Environment:
  VAULTPLAY_REBUILD=1   Force podman build
  VAULTPLAY_IMAGE       Image name (default: localhost/vaultplay:latest)
  VAULTPLAY_NO_OPEN=1   Do not open a browser
EOF
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

pick_port() {
  local p
  for p in $(seq "$PORT_START" "$PORT_END"); do
    # /dev/tcp connect success => something is already listening
    if (echo >/dev/tcp/127.0.0.1/"$p") >/dev/null 2>&1; then
      continue
    fi
    echo "$p"
    return 0
  done
  echo "no free port in ${PORT_START}-${PORT_END}" >&2
  exit 1
}

ensure_image() {
  if [[ "${VAULTPLAY_REBUILD:-0}" == "1" ]] || ! podman image exists "$IMAGE" >/dev/null 2>&1; then
    echo "Building $IMAGE ..."
    podman build -t "$IMAGE" -f "$SCRIPT_DIR/Containerfile" "$SCRIPT_DIR"
  else
    echo "Using existing image $IMAGE (set VAULTPLAY_REBUILD=1 to rebuild)"
  fi
}

test_isolation() {
  need_cmd podman
  echo "== Isolation test: --network none must block outbound HTTP =="
  set +e
  out="$(podman run --rm \
    --network none \
    --read-only \
    --cap-drop=ALL \
    --security-opt=no-new-privileges \
    --tmpfs /tmp:rw,noexec,nosuid,size=16m \
    docker.io/library/alpine:3.20 \
    wget -T 3 -O- https://example.com 2>&1)"
  rc=$?
  set -e
  echo "$out"
  if [[ $rc -eq 0 ]]; then
    echo "FAIL: outbound request unexpectedly succeeded" >&2
    exit 1
  fi
  echo "PASS: outbound request failed (exit $rc) under --network none"
}

cleanup() {
  local ec=$?
  if [[ -n "${SOCAT_PID:-}" ]] && kill -0 "$SOCAT_PID" 2>/dev/null; then
    kill "$SOCAT_PID" 2>/dev/null || true
    wait "$SOCAT_PID" 2>/dev/null || true
  fi
  if [[ -n "${CONTAINER_NAME:-}" ]]; then
    podman stop -t 2 "$CONTAINER_NAME" >/dev/null 2>&1 || true
  fi
  if [[ -n "${SOCKDIR:-}" && -d "${SOCKDIR}" ]]; then
    rm -rf "$SOCKDIR"
  fi
  exit "$ec"
}

wait_ready() {
  local port="$1" i
  for i in $(seq 1 60); do
    if curl -fsS "http://127.0.0.1:${port}/api/health" >/tmp/vaultplay-health.$$ 2>/dev/null; then
      cat /tmp/vaultplay-health.$$
      rm -f /tmp/vaultplay-health.$$
      return 0
    fi
    sleep 0.25
  done
  echo "timed out waiting for http://127.0.0.1:${port}/api/health" >&2
  exit 1
}

main() {
  if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    usage
    exit 0
  fi
  if [[ "${1:-}" == "--test-isolation" ]]; then
    test_isolation
    exit 0
  fi
  if [[ $# -lt 1 ]]; then
    usage >&2
    exit 1
  fi

  need_cmd podman
  need_cmd socat
  need_cmd curl

  local target
  target="$(realpath -e "$1")"

  local media_host only_rel=""
  if [[ -f "$target" ]]; then
    case "${target,,}" in
      *.mp4) ;;
      *)
        echo "file must be an .mp4: $target" >&2
        exit 1
        ;;
    esac
    media_host="$(dirname "$target")"
    only_rel="$(basename "$target")"
  elif [[ -d "$target" ]]; then
    media_host="$target"
  else
    echo "not a file or directory: $target" >&2
    exit 1
  fi

  ensure_image

  local run_id port
  run_id="$(date +%s)-$$"
  CONTAINER_NAME="vaultplay-${run_id}"
  SOCKDIR="${XDG_RUNTIME_DIR:-/tmp}/vaultplay-${run_id}"
  mkdir -p "$SOCKDIR"
  chmod 700 "$SOCKDIR"
  port="$(pick_port)"

  trap cleanup EXIT INT TERM

  local args=(
    run --rm -d
    --name "$CONTAINER_NAME"
    --network none
    --read-only
    --cap-drop=ALL
    --security-opt=no-new-privileges
    --userns=keep-id
    --user "$(id -u):$(id -g)"
    --tmpfs /tmp:rw,noexec,nosuid,size=64m
    -v "${media_host}:/videos:ro,Z"
    -v "${SOCKDIR}:/run/vaultplay:rw,Z"
    "$IMAGE"
    -root /videos
    -listen unix:///run/vaultplay/http.sock
  )
  if [[ -n "$only_rel" ]]; then
    args+=(-only "$only_rel")
  fi

  echo "Starting container $CONTAINER_NAME (network=none, media RO) ..."
  podman "${args[@]}" >/dev/null

  # Wait for the unix socket to appear
  local i
  for i in $(seq 1 80); do
    [[ -S "$SOCKDIR/http.sock" ]] && break
    sleep 0.1
  done
  if [[ ! -S "$SOCKDIR/http.sock" ]]; then
    echo "socket not created; container logs:" >&2
    podman logs "$CONTAINER_NAME" >&2 || true
    exit 1
  fi

  socat "TCP-LISTEN:${port},bind=127.0.0.1,reuseaddr,fork" "UNIX-CONNECT:${SOCKDIR}/http.sock" &
  SOCAT_PID=$!

  echo "Bridged 127.0.0.1:${port} -> unix://${SOCKDIR}/http.sock"
  echo "Health:"
  wait_ready "$port"
  echo

  local url="http://127.0.0.1:${port}/"
  echo "UI: $url"
  if [[ "${VAULTPLAY_NO_OPEN:-0}" != "1" ]] && command -v xdg-open >/dev/null 2>&1; then
    xdg-open "$url" >/dev/null 2>&1 || true
  fi

  echo "Press Ctrl+C to stop."
  while kill -0 "$SOCAT_PID" 2>/dev/null && podman container exists "$CONTAINER_NAME" 2>/dev/null; do
    sleep 1
  done
}

main "$@"
