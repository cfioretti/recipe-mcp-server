FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod ./
COPY go.sum ./
COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/recipe-mcp-server ./cmd

FROM alpine:3.21

RUN apk --no-cache add ca-certificates wget

WORKDIR /app

COPY --from=builder /out/recipe-mcp-server /app/recipe-mcp-server

EXPOSE 8080

ENTRYPOINT ["/app/recipe-mcp-server"]
