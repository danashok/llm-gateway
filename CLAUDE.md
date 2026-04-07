# LLM Gateway - Project Rules for Claude

## Project Overview
This is a Go-based LLM Gateway that provides a unified API for multiple LLM providers with compliance checking, rate limiting, and audit logging.

## Tech Stack
- Go 1.26
- Gin (HTTP framework)
- GORM (ORM for PostgreSQL)
- Redis (rate limiting, sessions, credential pool)
- Zap (structured logging)
- Swaggo (Swagger docs)

## Architectural Rules

### Function Signatures
- **ALWAYS** pass `ctx context.Context` as the first parameter
- **ALWAYS** pass `logger *zap.Logger` as the second parameter
- This applies to ALL functions in repositories, services, and handlers

### Interface Naming
- Use `IStructName` convention for interfaces (e.g., `ISessionRepository`, `IAPIKeyService`)

### Controller Pattern
- Controllers hold service interfaces and logger
- Controllers bind routes via `BindRoutes(rg *gin.RouterGroup)` method
- Routes are defined in controllers, not directly in router.go

## Global Rules - IMPORTANT

### Rule 1: OpenAPI Specification Updates
When ANY of the following changes occur, you MUST update `openapi/api.yaml`:
- New API endpoint is added
- Existing API endpoint is modified (path, method, parameters)
- Request/response schema changes
- New model/schema is added
- Authentication or security changes
- Error response changes

The OpenAPI file should include:
- Endpoint path and HTTP method
- Request body schema with examples
- Response schemas for all status codes (200, 400, 401, 403, 404, 422, 429, 500)
- Parameter descriptions
- Security requirements

### Rule 2: Technical Specification Updates
When ANY of the following changes occur, you MUST update `techspec.md`:
- New API endpoint is added (update API Endpoints table)
- Authentication model changes
- New database tables/models added
- Redis key patterns change
- Compliance checkers added/modified
- Configuration options added
- Architecture changes

The techspec should document:
- API endpoint tables organized by category
- Data flow diagrams (when architecture changes)
- Database schema changes
- Redis key patterns
- Configuration environment variables

## File Locations
- OpenAPI spec: `openapi/api.yaml`
- Technical spec: `techspec.md`
- Controllers: `internal/controller/`
- Services: `internal/service/`
- Repositories: `internal/repository/`
- Models: `internal/models/`
- Middleware: `internal/middleware/`

## Checklist Before Completing API Changes
- [ ] Controller created/updated with godoc comments
- [ ] Service interface and implementation updated
- [ ] Repository interface and implementation updated (if DB changes)
- [ ] `openapi/api.yaml` updated with new/modified endpoints
- [ ] `techspec.md` updated with new endpoints and any architectural changes
- [ ] `go build ./...` passes without errors
