# recipe-mcp-server

Minimal Go skeleton for the PizzaMaker MCP server.

## Endpoints

- `GET /health`: liveness/readiness check
- `GET /metrics`: minimal Prometheus-compatible metric
- `GET /mcp`: placeholder endpoint until MCP tool contract is defined

## Run locally

```bash
go run ./cmd
```

Default port is `8080`. Override with:

```bash
MCP_SERVER_PORT=8085 go run ./cmd
```

Application version in `/health` is read from `APP_VERSION` (default `dev`).
