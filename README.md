# v2home

A lightweight, self-hosted web panel for managing proxy tunnels — an ideological
analogue of [v2rayA](https://github.com/v2rayA/v2rayA), built as a pet project.

This repository is a **PoC scaffold**: a Go backend exposes a status API and a
WebSocket stream, and a Vue 3 + TypeScript + Tailwind frontend renders it. The
Vue bundle is embedded into the Go binary with `go:embed`, so the final artifact
is a single static executable that runs on OpenWrt 24.10 and other Linux systems.

## Stack

| Layer     | Technology                                      |
|-----------|-------------------------------------------------|
| Backend   | Go 1.23, `net/http`, `log/slog`, stdlib only    |
| Frontend  | Vue 3, TypeScript, Vite, Tailwind CSS           |
| Transport | REST (`/api/*`) + WebSocket (`/ws/status`)      |
| Packaging | `go:embed`, multi-stage Docker, static binary   |
| Target    | OpenWrt 24.10 (musl, procd), Linux amd64/arm64/mips |

## Repository layout

```text
.
├── backend/
│   ├── cmd/v2home/            # entrypoint
│   ├── internal/
│   │   ├── api/               # HTTP router, REST handlers, WS handler
│   │   ├── buildinfo/         # version/commit/build time (ldflags)
│   │   ├── config/            # flags + env
│   │   ├── status/            # runtime snapshot collector
│   │   └── wsproto/           # minimal RFC 6455 server implementation
│   └── webui/                 # go:embed + SPA handler; dist/ is the Vue build
├── frontend/
│   └── src/                   # Vue 3 + TS + Tailwind app
├── deploy/
│   ├── docker/Dockerfile      # multi-stage production build
│   └── openwrt/               # OpenWrt rootfs test image, init script, UCI config
├── scripts/openwrt-test.sh    # smoke test inside OpenWrt 24.10 userland
├── docker-compose.yml         # dev environment (Go + Node)
└── Makefile                   # single entrypoint for all Docker workflows
```

## Requirements

Only Docker is required on the host. Go and Node.js are **not** installed
locally — every build and test runs in a container.

## Quick start (development)

```sh
make dev
```

- Frontend dev server: <http://localhost:5173> (Vite HMR, proxies `/api` and `/ws`)
- Backend API: <http://localhost:8080/api/status>

Stop with `make down`.

## Production build

```sh
make build
docker run --rm -p 8080:8080 v2home:0.1.0
```

The multi-stage build compiles the Vue bundle, embeds it into the Go binary and
produces a small Alpine runtime image. `GOARCH` defaults to `amd64`; override it
with `make build GOARCH=arm64`. Open <http://localhost:8080>.

To build the frontend only (writes into `backend/webui/dist`):

```sh
make frontend-build
```

## OpenWrt 24.10

### Smoke test in an OpenWrt userland container

```sh
make openwrt-test
```

The script builds a static `linux/amd64` binary, wraps it into an
`openwrt/rootfs:x86-64-24.10.0` image and checks `/api/healthz` and
`/api/status` from inside the container. Override the release with
`OPENWRT_TAG=x86-64-24.10.1 make openwrt-test`.

### Cross-compile for a router

```sh
make build-openwrt GOARCH=mipsle GOMIPS=softfloat   # many MIPS routers
make build-openwrt GOARCH=arm GOARM=7               # armv7
make build-openwrt GOARCH=arm64                     # aarch64
make build-openwrt GOARCH=amd64                     # x86-64
```

The static binary lands in `dist/<arch>/v2home`.

### Manual install on a device

```sh
scp dist/mipsle/v2home root@192.168.1.1:/usr/sbin/v2home
scp deploy/openwrt/files/etc/init.d/v2home root@192.168.1.1:/etc/init.d/v2home
scp deploy/openwrt/files/etc/config/v2home root@192.168.1.1:/etc/config/v2home

ssh root@192.168.1.1
chmod +x /usr/sbin/v2home /etc/init.d/v2home
/etc/init.d/v2home enable
/etc/init.d/v2home start
logread -f | grep v2home
```

The service is managed by procd; settings live in `/etc/config/v2home`:

```sh
uci set v2home.main.listen='0.0.0.0:8080'
uci set v2home.main.log_level='debug'
uci commit v2home
/etc/init.d/v2home restart
```

## API

| Method | Path           | Description                              |
|--------|----------------|------------------------------------------|
| GET    | `/api/healthz` | Liveness probe                           |
| GET    | `/api/status`  | Full status snapshot (JSON)              |
| GET    | `/api/version` | Build metadata                           |
| GET    | `/ws/status`   | WebSocket stream of status snapshots     |

Example:

```sh
curl -s http://localhost:8080/api/status | jq
```

```json
{
  "app": "v2home",
  "version": "0.1.0",
  "commit": "unknown",
  "buildTime": "2026-10-27T12:00:00Z",
  "goVersion": "go1.23.x",
  "os": "linux",
  "arch": "amd64",
  "hostname": "router",
  "startedAt": "2026-10-27T12:00:00Z",
  "uptimeSeconds": 42,
  "now": "2026-10-27T12:00:42Z",
  "runtime": {
    "goroutines": 6,
    "numCpu": 4,
    "heapAllocBytes": 1234567,
    "heapSysBytes": 2345678,
    "gcCount": 3
  },
  "tunnels": { "total": 0, "running": 0 }
}
```

The WebSocket sends the same JSON object immediately after the handshake and
then every `--ws-interval` (default `2s`). The frontend reconnects with
exponential backoff.

## Configuration

Flags (env fallbacks in parentheses):

| Flag                 | Default            | Description                          |
|----------------------|--------------------|--------------------------------------|
| `--listen`           | `0.0.0.0:8080`     | HTTP listen address (`V2HOME_LISTEN`) |
| `--log-level`        | `info`             | `debug`, `info`, `warn`, `error` (`V2HOME_LOG_LEVEL`) |
| `--ws-interval`      | `2s`               | Status push interval (`V2HOME_WS_INTERVAL`) |
| `--shutdown-timeout` | `10s`              | Graceful shutdown timeout            |
| `--version`          | —                  | Print version and exit               |

## Tests

```sh
make test   # go test ./...
make vet    # go vet ./...
```

Tests cover the status API, the SPA fallback and the WebSocket framing
(handshake key, masked client frames, server frames).

## Notes and next steps

- `backend/webui/dist` is intentionally almost empty in git (`.gitkeep`). The
  Go server detects a missing `index.html` and serves a built-in placeholder
  page, so `go run` works before the first frontend build.
- The WebSocket implementation in `internal/wsproto` is a deliberately small
  RFC 6455 subset (text frames, ping/pong, close, fragmentation). It is enough
  for the PoC; swap it for `github.com/coder/websocket` or `gorilla/websocket`
  when the tunnel engine needs binary frames or compression.
- `tunnels` in the status payload is a placeholder for the future tunnel
  manager (xray/sing-box core, subscriptions, routing rules).
- The OpenWrt package (`opkg`) is not wired yet; the current flow installs the
  static binary and the procd init script manually. A `golang-package.mk`
  based recipe can be added once the binary stabilises.
