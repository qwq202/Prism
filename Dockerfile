# Author: ProgramZmh
# License: Apache-2.0
# Description: Dockerfile for chatnio

FROM golang:1.25-alpine AS backend

WORKDIR /backend
COPY . .

# Set go proxy to https://goproxy.cn (open for vps in China Mainland)
# RUN go env -w GOPROXY=https://goproxy.cn,direct
ARG TARGETARCH
ARG TARGETOS
ENV GOOS=$TARGETOS GOARCH=$TARGETARCH GO111MODULE=on CGO_ENABLED=1

# Install build dependencies
RUN apk update && \
    apk add --no-cache \
    gcc \
    musl-dev \
    g++ \
    make \
    linux-headers

# Build backend
RUN go build -o chat -a -ldflags="-extldflags=-static" .

FROM node:22-alpine AS frontend

WORKDIR /app
COPY ./app/package.json ./app/pnpm-lock.yaml ./app/pnpm-workspace.yaml ./

RUN corepack enable && \
    corepack prepare pnpm@11.0.3 --activate && \
    pnpm install --frozen-lockfile

COPY ./app .

RUN pnpm run build && \
    rm -rf node_modules src


FROM alpine

# Install dependencies
RUN apk upgrade --no-cache && \
    apk add --no-cache wget ca-certificates tzdata su-exec && \
    (update-ca-certificates 2>/dev/null || true) && \
    addgroup -S chat && \
    adduser -S -G chat chat && \
    mkdir -p /config /logs /storage /db && \
    chown -R chat:chat /config /logs /storage /db

# Set timezone
RUN echo "Asia/Shanghai" > /etc/timezone && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

WORKDIR /

# Copy dist
COPY --from=backend /backend/chat /chat
RUN ln -sf /chat /usr/local/bin/prism && \
    ln -sf /chat /usr/local/bin/chat
COPY --from=backend /backend/config.example.yaml /config.example.yaml
COPY --from=backend /backend/utils/templates /utils/templates
COPY --from=frontend --chown=chat:chat /app/dist /app/dist
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

# Volumes
VOLUME ["/config", "/logs", "/storage", "/db"]

# Expose port
EXPOSE 8094

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD sh -c 'wget -qO- "http://127.0.0.1:${SERVER_PORT:-8094}/health" >/dev/null || exit 1'

# Run application
ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["./chat"]
