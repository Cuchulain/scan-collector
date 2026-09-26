FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/scan-collector .

FROM alpine:3.22
RUN addgroup -S app && adduser -S -G app app && mkdir -p /data && chown app:app /data
COPY --from=build /out/scan-collector /usr/local/bin/scan-collector
USER app
ENV PORT=8765 DATA_DIR=/data
EXPOSE 8765
VOLUME ["/data"]
ENTRYPOINT ["scan-collector"]
