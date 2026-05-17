FROM golang:1.26.3-alpine AS builder
WORKDIR /usr/src/coinmon
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -v -o coinmon cmd/app/main.go

FROM --platform=$BUILDPLATFORM alpine:3.21
WORKDIR /usr/local/bin/
RUN apk add --no-cache tzdata && \
    addgroup -g 1000 coinmon && \
    adduser -u 1000 -G coinmon -s /sbin/nologin -D coinmon
ENV TZ=Asia/Tbilisi
COPY --from=builder /usr/src/coinmon/coinmon /usr/local/bin/coinmon
COPY --from=builder /usr/src/coinmon/web ./web
RUN chown -R coinmon:coinmon /usr/local/bin/web
USER coinmon
CMD ["coinmon"]