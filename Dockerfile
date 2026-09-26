FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
COPY main.go auth.go i18n.js ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/scan-collector .

FROM alpine:3.22
RUN apk add --no-cache su-exec && addgroup -S app && adduser -S -G app app && mkdir -p /data && chown app:app /data
COPY --from=build /out/scan-collector /usr/local/bin/scan-collector
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh
ENV PORT=8765 DATA_DIR=/data
EXPOSE 8765
VOLUME ["/data"]
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
