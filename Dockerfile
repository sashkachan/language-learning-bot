
FROM golang:1.21 as builder
WORKDIR /app
COPY cmd cmd
COPY pkg pkg
COPY go.mod go.sum .
RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags "-linkmode external -extldflags -static" -o langekko ./cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/langekko .
COPY scripts scripts
COPY templates templates
CMD ["./langekko", "telegram"]
