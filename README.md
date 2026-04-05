# llm-gateway

Production-grade internal LLM proxy gateway for an aerospace company. Provides a unified API endpoint for internal developers to access multiple LLM providers (internal hosted, OpenAI, Anthropic, Vertex AI, Bedrock, open source models) with compliance enforcement, rate limiting, cost control, and full audit logging.

## Tech Stack

- **Language**: Go 1.22+
- **HTTP Framework**: Gin
- **ORM**: GORM + PostgreSQL
- **Cache/Rate Limiting**: Redis
- **Logger**: Zap
- **Config**: env + godotenv
- **OpenAPI**: Swaggo
- **Metrics**: Prometheus

## Project Structure

```
llm-gateway/
├── cmd/
│   └── main.go                     # Application entry point
├── docker-compose/
│   ├── infra.yml                   # Database (Postgres) + Redis composition
│   └── local.yml                   # Go App container composition
├── openapi/                        # OpenAPI / Swagger output directory
├── internal/
│   ├── config/                     # Configuration management
│   ├── db/                         # Database initialization
│   ├── middleware/                 # HTTP middleware (auth, metrics)
│   ├── models/                     # Domain models (session, audit_log, team_budget, compliance_violation)
│   ├── repository/                 # Data access layer
│   ├── service/                    # Business logic layer
│   ├── compliance/                 # Compliance checking (secrets, PII, project policies)
│   ├── provider/                   # LLM provider adapters (internal, OpenAI, Anthropic)
│   ├── ratelimit/                  # Redis-based rate limiting
│   ├── worker/                     # Bounded goroutine worker pool
│   ├── gateway/                    # Main chat handler with SSE streaming
│   └── routes/                     # Route definitions
├── .env.example                    # Environment variables template
├── Dockerfile                      # Multi-stage Docker build
├── go.mod                          # Go module definition
├── Makefile                        # Build and run commands
└── README.md
```

## Architecture

### Request Flow

1. **Authentication** - Extract team/user context from Authorization header
2. **Rate Limiting** - Check Redis token bucket for (team_id, model_id)
3. **Session Management** - Load or create conversation session
4. **Compliance Check** - Parallel fan-out to all compliance checkers (secrets, PII, project policies)
5. **Provider Routing** - Route to appropriate LLM adapter based on model_id
6. **SSE Streaming** - Stream response chunks back to client
7. **Async Processing** - Update session history and record usage costs

### Compliance Checkers

- **Secret Checker** - Regex + entropy-based detection for API keys, tokens, passwords
- **PII Checker** - Detection for SSN, credit cards, emails, phone numbers, etc.
- **Project Checker** - Aerospace-specific sensitive content detection (ITAR, classified, export control)

### Provider Adapters

- **Internal** - Fully implemented for internally hosted models
- **OpenAI** - Stub (TODO: implement API integration)
- **Anthropic** - Stub (TODO: implement API integration)

## Getting Started

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- Make

### Setup

1. Copy the environment file:
   ```bash
   cp .env.example .env
   ```

2. Start the infrastructure (PostgreSQL + Redis):
   ```bash
   make db-up
   # or
   docker-compose -f docker-compose/infra.yml up -d
   ```

3. Run the application locally:
   ```bash
   make run
   # or
   go run cmd/main.go
   ```

### Running with Docker

Start both infrastructure and application:
```bash
make db-up
make app-up
# or
docker-compose -f docker-compose/infra.yml up -d
docker-compose -f docker-compose/local.yml up -d
```

### API Documentation

Generate Swagger documentation:
```bash
make swag
# or
swag init -g cmd/main.go -o openapi/ --parseDependency --parseInternal
```

Access Swagger UI at: `http://localhost:8080/swagger/index.html`

## Available Commands

| Command | Description |
|---------|-------------|
| `make build` | Build the application binary |
| `make run` | Run the application locally |
| `make test` | Run tests |
| `make clean` | Remove build artifacts |
| `make swag` | Generate Swagger documentation |
| `make db-up` | Start PostgreSQL and Redis containers |
| `make db-down` | Stop PostgreSQL and Redis containers |
| `make app-up` | Start application container |
| `make app-down` | Stop application container |

## API Endpoints

### Health & Metrics
- `GET /api/v1/health` - Health check endpoint
- `GET /metrics` - Prometheus metrics endpoint
- `GET /swagger/*` - Swagger documentation

### Chat (Protected - requires Authorization header)
- `POST /api/v1/chat` - Send a chat message (SSE streaming response)
- `GET /api/v1/models` - List available LLM models

### Chat Request Example

```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "model_id": "internal",
    "prompt": "Hello, how are you?",
    "session_id": "optional-uuid"
  }'
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Application port | `8080` |
| `DB_DRIVER` | Database driver | `postgres` |
| `DB_SOURCE` | Database connection string | See `.env.example` |
| `REDIS_ADDR` | Redis address | `localhost:6379` |
| `REDIS_PASSWORD` | Redis password | `` |
| `REDIS_DB` | Redis database number | `0` |
| `WORKER_POOL_SIZE` | Worker pool concurrency | `500` |
| `REQUEST_TIMEOUT` | Request timeout duration | `60s` |
| `SESSION_TTL` | Session time-to-live | `2h` |
| `DEFAULT_DAILY_BUDGET` | Default daily token budget | `100000` |
| `OPENAI_API_KEY` | OpenAI API key | `` |
| `ANTHROPIC_API_KEY` | Anthropic API key | `` |
| `INTERNAL_MODEL_URL` | Internal LLM service URL | `http://internal-llm:8000` |

## Prometheus Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `llm_gateway_requests_total` | Counter | team_id, model_id, status | Total requests |
| `llm_gateway_compliance_violations_total` | Counter | checker_name, severity | Compliance violations |
| `llm_gateway_provider_latency_seconds` | Histogram | model_id | Provider latency |
| `llm_gateway_tokens_used_total` | Counter | team_id, model_id, token_type | Token usage |
| `llm_gateway_active_requests` | Gauge | - | Currently active requests |
| `llm_gateway_request_duration_seconds` | Histogram | method, path, status | Request duration |

## Domain Models

- **Session** - Conversation session with message history
- **AuditLog** - Comprehensive audit trail for all requests
- **TeamBudget** - Token budget allocation and usage tracking per team
- **ComplianceViolation** - Record of detected compliance violations
