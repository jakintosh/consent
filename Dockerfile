FROM golang:1.24-alpine AS build

RUN apk add --no-cache ca-certificates git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go generate ./... \
	&& CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/consent ./cmd/consent

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
	&& addgroup -S -g 10001 consent \
	&& adduser -S -D -H -u 10001 -G consent consent \
	&& mkdir -p /config /data \
	&& chown -R consent:consent /config /data

COPY --from=build /out/consent /usr/local/bin/consent

USER consent:consent
EXPOSE 8000

ENTRYPOINT ["/usr/local/bin/consent"]
CMD ["serve", "--config-dir", "/config", "--data-dir", "/data"]
