# syntax=docker/dockerfile:1.7

ARG NODE_VERSION=22
ARG GO_VERSION=1.23

# ---------- frontend ----------
FROM node:${NODE_VERSION}-alpine AS frontend
WORKDIR /src/frontend

# package*.json matches package.json even when no lockfile is committed yet.
COPY frontend/package*.json ./
RUN if [ -f package-lock.json ]; then \
      npm ci --no-audit --no-fund; \
    else \
      npm install --no-audit --no-fund; \
    fi

COPY frontend/ ./
RUN npm run build

# ---------- backend ----------
FROM golang:${GO_VERSION}-alpine AS backend
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG GOARM=
ARG GOMIPS=softfloat
ARG VERSION=0.1.0
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

WORKDIR /src/backend
COPY backend/ ./
RUN go mod download
COPY --from=frontend /src/backend/webui/dist ./webui/dist

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOARM=${GOARM} GOMIPS=${GOMIPS} \
    go build -buildvcs=false -trimpath \
      -ldflags "-s -w \
        -X v2home/internal/buildinfo.Version=${VERSION} \
        -X v2home/internal/buildinfo.Commit=${COMMIT} \
        -X v2home/internal/buildinfo.BuildTime=${BUILD_TIME}" \
      -o /out/v2home ./cmd/v2home

# ---------- export (static binary for OpenWrt) ----------
FROM scratch AS export
COPY --from=backend /out/v2home /v2home

# ---------- runtime ----------
FROM alpine:3.20 AS runtime
RUN apk add --no-cache ca-certificates tzdata \
 && addgroup -S v2home \
 && adduser -S -G v2home v2home

COPY --from=backend /out/v2home /usr/local/bin/v2home

USER v2home
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
  CMD wget -qO- http://127.0.0.1:8080/api/healthz >/dev/null || exit 1

ENTRYPOINT ["/usr/local/bin/v2home"]
CMD ["--listen", "0.0.0.0:8080"]
