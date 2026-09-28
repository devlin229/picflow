FROM golang:1.27.1-alpine AS builder

ARG GOPROXY=https://proxy.golang.org,direct

WORKDIR /src

COPY go.mod go.sum ./
RUN GOPROXY="$GOPROXY" go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/server \
    .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S -G app app \
    && mkdir -p /app/logs \
    && chown -R app:app /app

WORKDIR /app

COPY --from=builder --chown=app:app /out/server /app/server

USER app

EXPOSE 8080

ENTRYPOINT ["/app/server"]
CMD ["serve"]
