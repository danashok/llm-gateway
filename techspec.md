# LLM Gateway - Technical Specification

## Overview

The LLM Gateway provides a unified API endpoint for developers to interact with multiple LLM providers (OpenAI, Anthropic, etc.) while enforcing compliance, authorization, rate limiting, and cost controls.

## Architecture

```
Developer App                    LLM Gateway                         LLM Providers
     |                               |                                    |
     |  POST /api/v1/chat            |                                    |
     |  {model_id, prompt,           |                                    |
     |   session_id?}                |                                    |
     |------------------------------>|                                    |
     |                               |                                    |
     |                    +----------+----------+                         |
     |                    | 1. AUTH MIDDLEWARE  |                         |
     |                    |    Extract API key  |                         |
     |                    |    -> team_id       |                         |
     |                    |    -> user_id       |                         |
     |                    |    -> developer_id  |                         |
     |                    +----------+----------+                         |
     |                               |                                    |
     |                    +----------+----------+                         |
     |                    | 2. RATE LIMIT CHECK |                         |
     |                    |    Redis: tokens    |                         |
     |                    |    remaining?       |                         |
     |                    +----------+----------+                         |
     |                               |                                    |
     |                    +----------+----------+                         |
     |                    | 3. SESSION LOAD     |                         |
     |                    |    Redis: get/create|                         |
     |                    |    session with     |                         |
     |                    |    conversation     |                         |
     |                    |    history          |                         |
     |                    +----------+----------+                         |
     |                               |                                    |
     |                    +----------+----------+                         |
     |                    | 4. COMPLIANCE CHECK |                         |
     |                    |    (PARALLEL)       |                         |
     |                    |    +- SecretChecker |                         |
     |                    |    +- PIIChecker    |                         |
     |                    |    +- ProjectChecker|                         |
     |                    +----------+----------+                         |
     |                               |                                    |
     |                    +----------+----------+                         |
     |                    | 5. ROUTE TO PROVIDER|                         |
     |                    |    model_id ->      |                         |
     |                    |    OpenAI/Anthropic |                         |
     |                    |    + Acquire API key|                         |
     |                    +----------+----------+                         |
     |                               |                                    |
     |                               |  Chat(messages + prompt)           |
     |                               |----------------------------------->|
     |                               |                                    |
     |<------------------------------|<-----------------------------------|
     |   SSE Stream:                 |   Stream chunks                    |
     |   data: {"content": "..."}    |                                    |
     |   data: {"content": "..."}    |                                    |
     |   data: [DONE]                |                                    |
     |                               |                                    |
     |                    +----------+----------+                         |
     |                    | 6. POST-PROCESSING  |                         |
     |                    |    (ASYNC)          |                         |
     |                    |    +- Save messages |                         |
     |                    |    |  to session    |                         |
     |                    |    +- Audit log     |                         |
     |                    |    +- Update budget |                         |
     |                    |    +- Release cred  |                         |
     |                    +---------------------+                         |
```

## Authentication Model

API keys are bound to a specific combination of:
- **Team ID** - Organization/team the developer belongs to
- **Developer ID** - Individual developer identity
- **Release Unit** - Project/product name the developer is authorized to use

### Authentication Flow

```
1. Admin authorizes developer for release unit(s)
   POST /api/v1/release-units/authorize
   {team_id, developer_id, release_units: ["project-alpha", "project-beta"]}

2. Developer creates API key for specific release unit
   POST /api/v1/api-keys
   {team_id, developer_id, release_unit, ...}
   -> Returns: llmgw_<random_64_chars>

3. Developer uses API key in requests
   Authorization: Bearer llmgw_...

4. Gateway validates:
   - API key exists and is active
   - API key not expired
   - Developer still authorized for release unit
```

### Database Tables

**api_keys**
| Column | Type | Description |
|--------|------|-------------|
| id | uint | Primary key |
| key | string | SHA256 hash of API key |
| key_prefix | string | First 8 chars for identification |
| team_id | uuid | Team the key belongs to |
| developer_id | uuid | Developer who owns the key |
| release_unit | string | Project/release unit name |
| is_active | bool | Whether key is active |
| expires_at | timestamp | Optional expiration |
| last_used_at | timestamp | Last usage timestamp |

**team_developer_release_units**
| Column | Type | Description |
|--------|------|-------------|
| team_id | uuid | Team ID |
| developer_id | uuid | Developer ID |
| release_unit | string | Authorized release unit |
| is_active | bool | Whether authorization is active |

## Request Pipeline

### Step 1: Authentication
- Extract API key from `Authorization` header (supports `Bearer <key>` format)
- Look up hashed key in database
- Validate: key exists, is active, not expired
- Verify developer is still authorized for the release unit
- Extract `team_id`, `user_id`, `developer_id`, `release_unit`
- Failure: `401 Unauthorized` or `403 Forbidden` (unauthorized release unit)

### Step 2: Rate Limiting
- Check Redis for remaining tokens: `ratelimit:{team_id}:{model_id}`
- Token bucket with 24-hour window
- Failure: `429 Too Many Requests`

### Step 3: Session Management
- If `session_id` provided: Load existing session from Redis
- Session key format: `session:{developer_id}:{session_id}` (ensures isolation)
- If no session: Create new session with TTL
- Session contains conversation history (messages array)
- Failure: `404 Not Found` (invalid session_id)

### Step 4: Compliance Check (Parallel)
All checkers run concurrently using `errgroup.WithContext`:

| Checker | Purpose | Severity |
|---------|---------|----------|
| **SecretChecker** | Detect API keys, passwords, tokens via regex + entropy | Critical |
| **PIIChecker** | Detect SSN, credit cards, emails, phone numbers | High |
| **ProjectChecker** | Detect sensitive aerospace project codenames | Critical |

- If any checker returns violations above threshold: `422 Unprocessable Entity`
- Violations are stored in database for audit

### Step 5: Provider Routing
- Map `model_id` to provider (e.g., `gpt-4` -> OpenAI, `claude-3` -> Anthropic)
- Acquire API credential from pool (weighted round-robin via Redis sorted set)
- Blocked credentials (rate-limited) are skipped
- Failure: `400 Bad Request` (unsupported model)

### Step 6: LLM Request & Streaming
- Build request with conversation history + current prompt
- Stream response via SSE (Server-Sent Events)
- Response headers include `X-Trace-ID` and `X-Session-ID`

### Step 7: Post-Processing (Async)
Non-blocking operations after response:
- Append user message and assistant response to session
- Write audit log with token counts and duration
- Update team budget usage
- Release credential back to pool

## Redis Data Model

```
+-------------------------------------------------------------+
|                         Redis                                |
+-------------------------------------------------------------+
|  Rate Limiting (Strings)                                    |
|    ratelimit:team123:gpt-4 = 9500  (TTL: 24h)              |
|    tokens:daily:team123 = 500                               |
+-------------------------------------------------------------+
|  Sessions (Strings - JSON)                                  |
|    session:dev-abc:sess-123 = {"id":...,"messages":[...]}  |
|    session:dev-abc:sess-456 = {"id":...,"messages":[...]}  |
+-------------------------------------------------------------+
|  Credential Pool (Sorted Sets + Strings)                    |
|    credpool:openai = [(key1, 3), (key2, 5), (key3, 1)]     |
|    credpool:anthropic = [(keyA, 0), (keyB, 2)]             |
|    credpool:blocked:openai:a1b2c3d4 = "1" (TTL: 60s)       |
+-------------------------------------------------------------+
```

### Redis Key Patterns

| Key Pattern | Type | Purpose | TTL |
|-------------|------|---------|-----|
| `ratelimit:{team_id}:{model_id}` | String | Token bucket counter | 24h |
| `tokens:daily:{team_id}` | String | Daily usage counter | 24h |
| `session:{developer_id}:{session_id}` | String (JSON) | Session with messages | Configurable |
| `credpool:{provider}` | Sorted Set | API keys sorted by usage | None |
| `credpool:blocked:{provider}:{key_hash}` | String | Blocked credential flag | 60s default |

## Database Schema (PostgreSQL)

### Tables

- **sessions** - Session metadata (also cached in Redis)
- **audit_logs** - Full request/response audit trail
- **team_budgets** - Token limits and usage per team
- **compliance_violations** - Recorded compliance violations

## Compliance Checkers

### SecretChecker
Detects secrets using:
1. **Regex patterns**: AWS keys, GitHub tokens, private keys, etc.
2. **Entropy analysis**: High-entropy strings likely to be secrets

### PIIChecker
Detects personally identifiable information:
- Social Security Numbers (SSN)
- Credit card numbers (with Luhn validation)
- Email addresses
- Phone numbers

### ProjectChecker
Detects sensitive aerospace project codenames:
- Configurable list of restricted project names
- Case-insensitive matching

## Credential Pool

Implements weighted round-robin with automatic backoff:

1. **Acquire**: Get credential with lowest usage score (not blocked)
2. **Release**: Decrement usage score after request completes
3. **MarkExhausted**: Block credential for `retry_after` duration when rate-limited

This ensures:
- Even distribution of load across API keys
- Automatic failover when keys hit rate limits
- Recovery after backoff period expires

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `APP_PORT` | HTTP server port | 8080 |
| `DB_HOST` | PostgreSQL host | localhost |
| `DB_PORT` | PostgreSQL port | 5432 |
| `DB_NAME` | Database name | llm_gateway |
| `REDIS_ADDR` | Redis address | localhost:6379 |
| `OPENAI_API_KEYS` | Comma-separated OpenAI keys | - |
| `ANTHROPIC_API_KEYS` | Comma-separated Anthropic keys | - |
| `GATEWAY_SESSION_TTL` | Session expiration | 30m |
| `GATEWAY_DEFAULT_DAILY_BUDGET` | Default daily token limit | 100000 |

## API Endpoints

### Chat & Models
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/chat` | Send chat message, stream response |
| GET | `/api/v1/models` | List available models |

### Sessions
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/sessions` | Create new session |
| GET | `/api/v1/sessions` | List developer's sessions |
| GET | `/api/v1/sessions/{id}` | Get session by ID |
| DELETE | `/api/v1/sessions/{id}` | Delete session |

### API Key Management
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/api-keys` | Create new API key |
| GET | `/api/v1/api-keys` | List developer's API keys |
| DELETE | `/api/v1/api-keys/{id}` | Revoke an API key |

### Release Unit Authorization
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/release-units` | List authorized release units |
| POST | `/api/v1/release-units/authorize` | Authorize developer for release unit |
| DELETE | `/api/v1/release-units/revoke` | Revoke release unit authorization |

### Audit Logs
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/audit/trace/{trace_id}` | Get audit logs by trace |
| GET | `/api/v1/audit/team` | Get team's audit logs |
| GET | `/api/v1/audit/user` | Get user's audit logs |
| GET | `/api/v1/audit/range` | Get audit logs by date range |

### Budget Management
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/budget` | Get team budget |
| POST | `/api/v1/budget` | Create team budget |
| PUT | `/api/v1/budget` | Update team budget |
| GET | `/api/v1/budget/check` | Check if budget allows request |

### Compliance
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/compliance/trace/{trace_id}` | Get violations by trace |
| GET | `/api/v1/compliance/team` | Get team's violations |
| GET | `/api/v1/compliance/severity/{level}` | Get violations by severity |
| GET | `/api/v1/compliance/stats` | Get violation statistics |

### System
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/health` | Health check |
| GET | `/metrics` | Prometheus metrics |
| GET | `/swagger/*` | Swagger documentation |

## Error Responses

| Status | Meaning | When |
|--------|---------|------|
| 400 | Bad Request | Invalid JSON, unsupported model |
| 401 | Unauthorized | Missing/invalid API key |
| 404 | Not Found | Session not found |
| 422 | Unprocessable Entity | Compliance violation detected |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Provider error, system failure |

## Observability

### Prometheus Metrics
- `http_requests_total` - Request count by method, path, status
- `http_request_duration_seconds` - Request latency histogram
- `llm_tokens_total` - Token usage by team, model
- `compliance_violations_total` - Violation count by checker, severity

### Structured Logging (Zap)
All logs include:
- `trace_id` - Request correlation ID
- `team_id`, `user_id`, `developer_id` - Identity context
- `model_id` - Target model
- `duration_ms` - Processing time
