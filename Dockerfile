FROM --platform=$BUILDPLATFORM node:24-alpine AS frontend-builder

WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.27.1-alpine AS backend-builder

ARG GOPROXY=https://proxy.golang.org,direct
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN GOPROXY="$GOPROXY" go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/picflow \
    .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata font-noto-cjk \
    && addgroup -S app \
    && adduser -S -G app app \
    && mkdir -p /app/web /app/data /app/logs \
    && chown -R app:app /app

WORKDIR /app

COPY --from=backend-builder --chown=app:app /out/picflow /app/server
COPY --from=frontend-builder --chown=app:app /src/frontend/dist /app/web

USER app

ENV NAME=picflow \
    ENV=release \
    HOST=0.0.0.0 \
    PORT=8080 \
    DATA_DIR=/app/data \
    WEB_DIR=/app/web \
    MAX_UPLOAD_SIZE_MB=20 \
    WORKER_COUNT=2

VOLUME ["/app/data"]
EXPOSE 8080

ENTRYPOINT ["/app/server"]
CMD ["serve"]
