# Vaultplay

Browse and play local MP4 files inside a fully isolated **rootless Podman** container. The UI is a small Vue app served by a Go binary; the browser only talks to `127.0.0.1` on your machine.

## Quick start

```bash
./watch.sh /path/to/videos
# or a single file
./watch.sh /path/to/clip.mp4
```

First run builds `localhost/vaultplay:latest` with `podman build`. Force a rebuild with `VAULTPLAY_REBUILD=1`.

Requirements on the host: `podman`, `socat`, `curl`, and optionally `xdg-open`.

## What is isolated

| Control | Effect |
|---|---|
| `--network none` | Container has **no IP network**. It cannot reach the internet or LAN. |
| Unix socket + host `socat` | UI is reachable only as `http://127.0.0.1:<port>` on the host. |
| Media mount `:ro,Z` | Only the chosen folder/file tree is mounted, **read-only**. |
| `--read-only` | Container root filesystem is immutable. |
| `--cap-drop=ALL` | All Linux capabilities dropped. |
| `--security-opt=no-new-privileges` | No privilege escalation. |
| `--userns=keep-id` + `--user` | Runs as your UID inside a user namespace. |
| `--tmpfs /tmp` | Scratch space without touching the host disk layout. |
| No CDN assets | Frontend fonts/JS/CSS are baked into the image. |

## What is not isolated

- Your **host browser** and **host `socat`** are outside the container.
- The host process that launches Vaultplay can still read the same media files.
- Building the image needs network once (to pull base images and npm/Go modules). Runtime does not.

## Networking options (and why this one)

1. **`--network none` + Unix socket + `socat` on 127.0.0.1** (chosen)  
   Strongest isolation: no network namespace at all. A normal browser cannot open a Unix socket, so the host bridges loopback TCP → socket.

2. **`-p 127.0.0.1:PORT:PORT` with pasta/slirp4netns**  
   Easy for browsers, but the container still has an IP stack and can usually make outbound connections unless you add more controls.

3. **Internal Podman network / pasta tweaks**  
   Can reduce outbound reachability, but is weaker and more brittle than having no network.

## Isolation test

```bash
./watch.sh --test-isolation
```

This runs Alpine under the same `--network none` hardening and attempts `wget https://example.com`. Success of the test means the download **fails**.

When Vaultplay itself is running, `GET /api/health` includes an `outbound` probe (`1.1.1.1:443`). With `--network none` you should see `"reachable": false`.

## Development (optional, on the host)

```bash
# Frontend
cd web && npm install && npm run build && cd ..

# Backend (TCP for local debug)
go run . -root /path/to/videos -listen 127.0.0.1:8787
```

## API

- `GET /api/videos` — indexed MP4 list (`name`, `relPath`, `dir`, `size`, `durationMs`)
- `GET /api/health` — readiness + outbound probe
- `GET /media/<relPath>` — MP4 stream with HTTP Range support
