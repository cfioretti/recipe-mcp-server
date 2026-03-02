# recipe-mcp-server

Minimal Go skeleton for the PizzaMaker MCP server.

## Endpoints

- `GET /health`: liveness/readiness check
- `GET /metrics`: Prometheus-compatible metrics endpoint
- `GET /mcp`: MCP server info
- `GET /mcp/tools`: list available tools and schemas
- `POST /mcp/tools/generate_recipe`: generate recipe draft (`mode`, `prompt`, optional `constraints`)
- `POST /mcp/tools/customize_recipe`: customize recipe draft (`mode`, `prompt`, optional `constraints`, `baseRecipe`)

## Run locally

```bash
go run ./cmd
```

Default port is `8080`. Override with:

```bash
MCP_SERVER_PORT=8085 go run ./cmd
```

Application version in `/health` is read from `APP_VERSION` (default `dev`).

## AI Provider Configuration

The service supports multiple provider backends through environment variables:

- `AI_PROVIDER`: `ollama` | `external` | `mock` (default: `ollama`)
- `AI_HTTP_TIMEOUT_MS`: request timeout in milliseconds (default: `60000`)
- `AI_GENERATION_MAX_ATTEMPTS`: max retry attempts when output is invalid (default: `2`)
- `OLLAMA_BASE_URL`: Ollama URL (default: `http://localhost:11434`)
- `OLLAMA_MODEL`: model name used for generation (default: `llama3.2:3b`)
- `EXTERNAL_API_BASE_URL`: external provider base URL (required for `external`)
- `EXTERNAL_API_KEY`: optional bearer token for `external`
