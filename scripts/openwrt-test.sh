#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT"

GOARCH=${GOARCH:-amd64}
OPENWRT_TAG=${OPENWRT_TAG:-x86-64-24.10.0}
IMAGE=${IMAGE:-v2home-openwrt:test}
CONTAINER=${CONTAINER:-v2home-openwrt-test}
PORT=${PORT:-8080}

if [ "$GOARCH" != "amd64" ]; then
  echo "openwrt-test currently supports GOARCH=amd64 only (got $GOARCH)" >&2
  exit 1
fi

echo "==> building static binary for linux/$GOARCH"
rm -rf "dist/$GOARCH"
docker build \
  --build-arg TARGETARCH="$GOARCH" \
  --target export \
  --output "type=local,dest=dist/$GOARCH" \
  -f deploy/docker/Dockerfile .

echo "==> building OpenWrt $OPENWRT_TAG test image"
mkdir -p deploy/openwrt/bin
cp "dist/$GOARCH/v2home" deploy/openwrt/bin/v2home
docker build \
  --build-arg OPENWRT_TAG="$OPENWRT_TAG" \
  -t "$IMAGE" \
  deploy/openwrt

echo "==> starting container"
docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
docker run -d --name "$CONTAINER" -p "$PORT:8080" "$IMAGE" >/dev/null

cleanup() {
  docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
}
trap cleanup EXIT
trap 'exit 130' INT TERM

echo "==> waiting for /api/healthz"
attempt=0
while [ "$attempt" -lt 30 ]; do
  if docker exec "$CONTAINER" wget -qO- "http://127.0.0.1:8080/api/healthz" >/dev/null 2>&1; then
    break
  fi
  attempt=$((attempt + 1))
  sleep 1
done

if [ "$attempt" -ge 30 ]; then
  echo "health check timed out; container logs:" >&2
  docker logs "$CONTAINER" >&2 || true
  exit 1
fi

echo "==> /api/status"
docker exec "$CONTAINER" wget -qO- "http://127.0.0.1:8080/api/status"
echo
echo "==> OpenWrt smoke test passed"
