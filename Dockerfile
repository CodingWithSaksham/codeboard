FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o codeboard ./cmd/api/main.go

# ── final image ───────────────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Kolkata

WORKDIR /app
COPY --from=builder /app/codeboard .

EXPOSE 8000
CMD ["./codeboard"]
