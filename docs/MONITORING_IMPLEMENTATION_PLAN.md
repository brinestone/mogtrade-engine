# Monitoring and Observability Implementation Plan for MogTrade

## Overview
This plan outlines the implementation of monitoring and observability using Prometheus and Grafana for the MogTrade trading platform, with a hybrid approach keeping Sentry for error tracking while adding Prometheus for metrics.

## Current State Analysis
- **Go version**: 1.26.6
- **Existing monitoring**: Sentry error tracking (v0.49.0)
- **Existing rate limiting**: Custom middleware using golang.org/x/time/rate
- **HTTP framework**: Gin-gonic v1.12.0
- **Modules**: web, core, infra, services
- **No existing Prometheus metrics**

## Implementation Strategy: Hybrid Approach
Keep Sentry for error tracking and issue management while adding Prometheus for:
- Time-series metrics
- Alerting based on thresholds
- Long-term trend analysis
- Infrastructure monitoring

## Phase 1: Foundation (Week 1)

### 1.1 Add Dependencies
```bash
# Add to app/go.mod
github.com/prometheus/client_golang v1.12.1
# prometheus/common and client_model are transitive dependencies
```

### 1.2 Initialize Prometheus in main.go
```go
package main

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/brinestone/mogtrade/web/api"
    // ... other imports
)

func main() {
    # ... existing code
    
    # Initialize metrics
    prometheus.MustRegister(
        api.HTTPRequests,
        api.HTTPRequestDuration,
        # ... other metrics
    )
    
    # ... rest of main function
}
```

### 1.4 Add /metrics Endpoint
New file: `app/api/metrics.go`
```go
package api

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/prometheus/client_golang/prometheus"
)

var HTTPRequests = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "mogtrade_http_requests_total",
        Help: "Total number of HTTP requests",
    },
    []string{"method", "endpoint", "status"},
)

var HTTPRequestDuration = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name:    "mogtrade_http_request_duration_seconds",
        Help:    "HTTP request duration in seconds",
        Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
    },
    []string{"method", "endpoint"},
)

func init() {
    prometheus.MustRegister(HTTPRequests, HTTPRequestDuration)
}

func MetricsEndpoint(c *gin.Context) {
    handler := prometheus.Handler().ServeHTTP
    handler(c.Writer, c.Request)
}
```

### 1.4 Add Metrics Middleware
New file: `app/middleware/metrics.go`
```go
package middleware

import (
    "time"
    
    "github.com/gin-gonic/gin"
    "github.com/brinestone/mogtrade/web/api"
)

func Metrics() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        duration := time.Since(start).Seconds()
        
        api.HTTPRequests.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
            string(c.Writer.Status()),
        ).Inc()
        
        api.HTTPRequestDuration.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
        ).Observe(duration)
    }
}
```

### 1.5 Update Router Registration
In `app/api/api.go`:
```go
func MountApiV1(ctx context.Context, r *gin.RouterGroup) error {
    # ... existing code
    
    # Add metrics middleware
    r.Use(middleware.Metrics())
    
    # Add metrics endpoint
    r.GET("/metrics", api.MetricsEndpoint)
    
    # ... rest
}
```

## Phase 2: Application Metrics (Week 2-3)

### 2.1 Business Metrics

#### Order Processing Metrics
New file: `services/metrics/order_metrics.go`
```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
)

var OrderProcessed = prometheus.NewCounter(
    prometheus.CounterOpts{
        Name: "mogtrade_orders_processed_total",
        Help: "Total number of orders processed",
    },
)

var OrderLatency = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name:    "mogtrade_order_latency_seconds",
        Help:    "Order processing latency in seconds",
        Buckets: prometheus.LinearBuckets(0.01, 0.01, 20), # 10ms to 1s
    },
    []string{"symbol"},
)

func init() {
    prometheus.MustRegister(OrderProcessed, OrderLatency)
}
```

#### Trade Execution Metrics
```go
var TradesExecuted = prometheus.NewCounter(
    prometheus.CounterOpts{
        Name: "mogtrade_trades_executed_total",
        Help: "Total number of trades executed",
    },
)

var TradeLatency = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name:    "mogtrade_trade_latency_seconds",
        Help:    "Trade execution latency in seconds",
        Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
    },
    []string{"symbol"},
)

func init() {
    prometheus.MustRegister(TradesExecuted, TradeLatency)
}
```

### 2.3 Authentication Metrics
```go
var LoginAttempts = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "mogtrade_login_attempts_total",
        Help: "Total number of login attempts",
    },
    []string{"result"}, # success|failure
)

var PasswordResetRequests = prometheus.NewCounter(
    prometheus.CounterOpts{
        Name: "mogtrade_password_reset_requests_total",
        Help: "Total number of password reset requests",
    },
)

var MFAEnrollments = prometheus.NewCounter(
    prometheus.CounterOpts{
        Name: "mogtrade_mfa_enrollments_total",
        Help: "Total number of MFA enrollments",
    },
)

func init() {
    prometheus.MustRegister(LoginAttempts, PasswordResetRequests, MFAEnrollments)
}
```

### 2.4 Database Metrics
In `infra/db/queries.go` or similar:
```go
var DBQueryDuration = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name:    "mogtrade_db_query_duration_seconds",
        Help:    "Database query duration in seconds",
        Buckets: prometheus.ExponentialBuckets(0.0001, 2, 15), # 0.1ms to 16s
    },
    []string{"operation"},
)

func init() {
    prometheus.MustRegister(DBQueryDuration)
}
```

### 2.4 Connection Pool Metrics
```go
var DBConnections = prometheus.NewGauge(
    prometheus.GaugeOpts{
        Name: "mogtrade_db_connections_active",
        Help: "Number of active database connections",
    },
)

var DBPoolWait = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name:    "mogtrade_db_pool_wait_seconds",
        Help:    "Database pool wait time in seconds",
        Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
    },
    []string{"pool"},
)

func init() {
    prometheus.MustRegister(DBConnections, DBPoolWait)
}
```

## Phase 3: Grafana Dashboards (Week 3-4)

### 3.1 HTTP Dashboard
- Requests per minute by endpoint
- Error rate by endpoint (rate of 5xx responses)
- Request duration percentiles (p50, p95, p99)
- Active requests gauge

### 3.2 Application Dashboard
- Orders processed per minute
- Trade execution latency distribution
- API response times by route
- Login attempt success/failure rate
- Password reset request rate
- MFA enrollment rate

### 3.3 Security Dashboard
- Failed login attempts over time
- Account lockout events
- Password change frequency
- MFA enrollment/disenrollment

### 3.4 Database Dashboard
- Active connections
- Query duration distribution
- Pool wait times
- Connection errors

## Phase 4: Alerting Rules (Week 4+)

### 4.1 Critical Alerts
- High error rate (>5% of requests returning 5xx)
- Unusual spike in failed login attempts
- Database connection pool exhaustion
- API latency exceeding thresholds

### 4.2 Warning Alerts
- Increasing request duration trend
- Rising password reset request rate
- MFA enrollment below target

### 4.3 Alert Channel Configuration
- Email notifications
- Slack/webhook integration
- PagerDuty for critical issues

## Sentry Integration Maintenance

### Keep Sentry For:
- Error tracking with stack traces
- User-friendly error reports
- Deployment tracking
- Issue management

### Use Prometheus For:
- Metrics and trend analysis
- Alerting and threshold monitoring
- Infrastructure health
- Business KPI tracking

### Hybrid Benefits:
- Sentry: "What error happened and when"
- Prometheus: "Is the system healthy and performing well"
- Combined: Complete observability picture

## Implementation Checklist

### High Priority (Must Have)
- [ ] Add prometheus client_golang dependency
- [ ] Initialize metrics in main.go
- [ ] Add /metrics endpoint
- [ ] Basic HTTP request metrics (counter + histogram)
- [ ] Metrics middleware integration
- [ ] Grafana dashboard setup with basic metrics

### Medium Priority (Should Have)
- [ ] Business metrics (orders, trades, auth)
- [ ] Database connection and query metrics
- [ ] Custom Grafana dashboards for each metric category
- [ ] Basic alerting rules (2-3 critical alerts)

### Low Priority (Nice to Have)
- [ ] OpenTelemetry tracing integration
- [ ] Advanced metric categories (rate, quantity)
- [ ] Custom exporters (Pushgateway, remote write)
- [ ] Full alerting suite (10+ alerts)
- [ ] Logging correlation with trace IDs
- [ ] Service mesh metrics (if applicable)

## Risk Assessment

### Low Risk
- Adding /metrics endpoint (no breaking changes)
- Adding middleware (non-invasive)
- New metric families (additive)

### Medium Risk
- Database query timing (minor performance impact, measure)
- Histogram bucket selection (affects usefulness, not correctness)

### Low Risk (Sentry keeping)
- Keeping Sentry during transition
- No immediate removal of existing error tracking

## Success Metrics

### Technical Success
- /metrics endpoint returns 200 with Prometheus format
- Grafana dashboard displays without errors
- No performance degradation >5%
- All existing tests pass

### Business Success
- Team can monitor system health via dashboards
- Alerts catch issues before users notice
- Historical data available for capacity planning
- Better understanding of system performance patterns

## Next Steps

1. **Immediate**: Add prometheus client_golang to go.mod
2. **This week**: Implement Phase 1 (foundation metrics + /metrics endpoint)
3. **Next week**: Implement Phase 2 (application-specific metrics)
4. **Week 3**: Set up Grafana and create initial dashboards
5. **Week 4**: Implement alerting rules and refine dashboards

## Code Ownership
- **API layer**: `app/api/` - Metrics endpoint and HTTP metrics
- **Middleware**: `app/middleware/` - Metrics middleware
- **Services**: `services/` - Business metrics (orders, trades, auth)
- **Infra**: `infra/` - Database metrics

## Dependencies Update

### app/go.mod additions:
```
github.com/prometheus/client_golang v1.12.1
# (prometheus/common, client_model transitive)
```

No other modules need immediate updates, but infra/go.mod and services/go.mod will automatically get the transitive dependencies.

## Rollback Plan

If issues arise:
1. Remove prometheus imports from main.go
2. Remove /metrics route from router
3. Remove middleware from router usage
4. Run `go mod tidy` to clean up go.mod
5. Verify application still functions with Sentry only

The rollback can be done in under 30 minutes since changes are additive and non-breaking.