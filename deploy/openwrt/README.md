# OpenWrt 24.10 notes

The Go binary is fully static (`CGO_ENABLED=0`), so it runs on OpenWrt's musl
userland without extra libraries.

## Smoke test in Docker

From the repository root:

```sh
make openwrt-test
```

This builds `linux/amd64`, copies it into `deploy/openwrt/bin/v2home`, builds an
image from `openwrt/rootfs:x86-64-24.10.0` and checks the API from inside the
container. Use another release with:

```sh
OPENWRT_TAG=x86-64-24.10.1 make openwrt-test
```

## Cross-compilation

```sh
make build-openwrt GOARCH=mipsle GOMIPS=softfloat
make build-openwrt GOARCH=arm GOARM=7
make build-openwrt GOARCH=arm64
```

The result is `dist/<arch>/v2home`.

## Manual installation on a router

```sh
scp dist/mipsle/v2home root@192.168.1.1:/usr/sbin/v2home
scp files/etc/init.d/v2home root@192.168.1.1:/etc/init.d/v2home
scp files/etc/config/v2home root@192.168.1.1:/etc/config/v2home

ssh root@192.168.1.1
chmod +x /usr/sbin/v2home /etc/init.d/v2home
/etc/init.d/v2home enable
/etc/init.d/v2home start
```

Check the service:

```sh
logread -f | grep v2home
wget -qO- http://127.0.0.1:8080/api/status
```

The panel listens on all interfaces by default. On a stock OpenWrt firewall the
LAN zone can reach it; for WAN access add an explicit firewall rule and put the
panel behind a reverse proxy with authentication.

## UCI options

`/etc/config/v2home`:

```uci
config v2home 'main'
	option listen '0.0.0.0:8080'
	option log_level 'info'
```

Apply changes with `/etc/init.d/v2home restart` (or `reload`, which triggers the
procd reload trigger).
