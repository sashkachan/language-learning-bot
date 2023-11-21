FROM golang:1.21 as builder

WORKDIR /app
COPY cmd cmd
COPY pkg pkg
COPY go.mod go.sum .
RUN ls -la .
RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o bot cmd/main.go 

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/bot .
CMD ["./bot"]