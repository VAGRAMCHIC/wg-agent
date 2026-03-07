# ---------- Stage 1: build ----------
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o wg-agent ./cmd/agent


# ---------- Stage 2: runtime ----------
FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/wg-agent .

EXPOSE 8050

CMD ["./wg-agent"]
