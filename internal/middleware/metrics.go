package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RequestsTotal counts total requests by team, model, and status
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_gateway_requests_total",
			Help: "Total number of requests to the LLM gateway",
		},
		[]string{"team_id", "model_id", "status"},
	)

	// ComplianceViolationsTotal counts compliance violations by checker and severity
	ComplianceViolationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_gateway_compliance_violations_total",
			Help: "Total number of compliance violations detected",
		},
		[]string{"checker_name", "severity"},
	)

	// ProviderLatency tracks latency to LLM providers
	ProviderLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "llm_gateway_provider_latency_seconds",
			Help:    "Latency of requests to LLM providers",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120},
		},
		[]string{"model_id"},
	)

	// TokensUsedTotal counts tokens used by team and model
	TokensUsedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_gateway_tokens_used_total",
			Help: "Total tokens used by team and model",
		},
		[]string{"team_id", "model_id", "token_type"},
	)

	// ActiveRequests tracks currently active requests
	ActiveRequests = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "llm_gateway_active_requests",
			Help: "Number of currently active requests",
		},
	)

	// RequestDuration tracks overall request duration
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "llm_gateway_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"method", "path", "status"},
	)
)

// TraceIDKey is the context key for trace ID
const TraceIDKey = "trace_id"

// MetricsMiddleware adds Prometheus metrics collection
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate and inject trace ID
		traceID := uuid.New().String()
		c.Set(TraceIDKey, traceID)
		c.Header("X-Trace-ID", traceID)

		// Track active requests
		ActiveRequests.Inc()
		defer ActiveRequests.Dec()

		// Track request duration
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		RequestDuration.WithLabelValues(c.Request.Method, c.FullPath(), status).Observe(duration)
	}
}

// GetTraceID retrieves the trace ID from the gin context
func GetTraceID(c *gin.Context) string {
	if traceID, exists := c.Get(TraceIDKey); exists {
		return traceID.(string)
	}
	return ""
}

// RecordRequest records a request metric
func RecordRequest(teamID, modelID string, status int) {
	RequestsTotal.WithLabelValues(teamID, modelID, strconv.Itoa(status)).Inc()
}

// RecordComplianceViolation records a compliance violation metric
func RecordComplianceViolation(checkerName, severity string) {
	ComplianceViolationsTotal.WithLabelValues(checkerName, severity).Inc()
}

// RecordProviderLatency records provider latency
func RecordProviderLatency(modelID string, duration time.Duration) {
	ProviderLatency.WithLabelValues(modelID).Observe(duration.Seconds())
}

// RecordTokensUsed records token usage
func RecordTokensUsed(teamID, modelID string, promptTokens, outputTokens int) {
	TokensUsedTotal.WithLabelValues(teamID, modelID, "prompt").Add(float64(promptTokens))
	TokensUsedTotal.WithLabelValues(teamID, modelID, "output").Add(float64(outputTokens))
}
