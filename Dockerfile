FROM golang:1.19-alpine AS builder
WORKDIR /usr/src/coinmon
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -v -o coinmon cmd/app/main.go

FROM --platform=$BUILDPLATFORM alpine:3.17.0 
WORKDIR /usr/local/bin/
RUN apk add --no-cache tzdata
ENV TZ=Europe/Moscow
COPY --from=builder /usr/src/coinmon/coinmon /usr/local/bin/coinmon
COPY --from=builder /usr/src/coinmon/web ./web
CMD ["coinmon"]