# GitHub Issues for Monitoring Implementation

## High Priority Issues

### [MON-001] Implement Basic Monitoring Infrastructure
- Add Prometheus client_golang dependency to app/go.mod
- Initialize Prometheus metrics in main.go
- Add /metrics endpoint with basic HTTP metrics
- Implement metrics middleware
- Create basic HTTP metrics (requests total, request duration)

### [MON-002] Implement Business Metrics
- Add order processing metrics (orders_processed_total, order_latency)
- Add trade execution metrics (trades_executed_total, trade_latency)
- Add authentication metrics (login_attempts, password_reset_requests, mfa_enrollments)
- Add database metrics (query_duration, connection_pool_wait)

### [MON-003] Implement Monitoring Middleware and Endpoint
- Create metrics middleware in app/middleware/
- Add metrics middleware to router
- Add /metrics endpoint to API routes
- Implement proper metric registration and initialization

### [MON-004] Create Initial Grafana Dashboards
- Set up Prometheus data source in Grafana
- Create HTTP overview dashboard (requests, errors, duration)
- Create application performance dashboard (orders, trades, API latency)
- Set up basic alerting rules for critical issues

### [MON-005] Implement Sentry/Prometheus Hybrid Approach
- Document the hybrid monitoring approach
- Define what metrics to track in Prometheus vs. errors in Sentry
- Create guidelines for when to use each tool
- Document the monitoring philosophy for the team

### [MON-006] Implement Security Monitoring
- Add failed login tracking and anomaly detection
- Implement session invalidation on password change
- Add MFA status tracking
- Create security-focused dashboards in Grafana

## Implementation Timeline
- **Week 1**: Complete Phase 1 (foundation metrics)
- **Week 2-3**: Implement Phase 2 (business metrics)
- **Week 4**: Set up Grafana dashboards and basic alerting
- **Week 5+**: Advanced monitoring features and optimization

## Ownership
- API layer: app/api/
- Middleware: app/middleware/
- Services: services/
- Infrastructure: infra/