# Asynq Background Job Integration Plan for MogTrade

## Overview
This plan outlines the integration of [Asynq](https://github.com/hibiken/asynq) - a fast, reliable, and lightweight distributed task queue for Go - into the MogTrade project to replace/ad supplement the current event bus-based approach for background job processing.

## 1. Asynq Basics

### 1.1 Key Features
- **Simple API**: Easy-to-use producer/consumer pattern
- **Reliability**: Built on PostgreSQL for task storage, ensuring durability
- **Visibility**: Built-in dashboard at `/q` for monitoring task execution
- **Retry & Timeout**: Configurable retry policy and timeout per task
- **Middleware**: Support for middleware for logging, metrics, etc.
- **Worker Pool**: Horizontal scaling via multiple worker processes

### 1.2 Core Concepts
- **Task**: A unit of work defined by a type name and payload
- **Producer**: Creates and enqueues tasks
- **Consumer/Worker**: Processes tasks from the queue
- **Broker**: PostgreSQL database that stores tasks

## 2. Asynq Integration Architecture

### 2.1 Component Diagram

```
+-----------------+     +------------------+
|   API Layer     |     |   Task Producer  |
| (GIN HTTP)      |     | (enqueue tasks)  |
+--------+--------+     +--------+---------+
         |                  |
         v                  v
+--------+--------+   +------------------+
|   Asynq Client  |   |   PostgreSQL     |
| (task.Enqueue)  |   |   (task queue)   |
+--------+--------+   +------------------+
         |                  |
         v                  v
+--------+--------+   +------------------+
|   Asynq Worker  |   |   Task Definitions|
| (task.Process)  |   | (task.Type + fn)  |
+--------+--------+   +------------------+
         |
         v
+------------------+
|   Task Dashboard |
|   (/q endpoint)  |
+------------------+
```

### 2.2 Integration Points with MogTrade

| MogTrade Component | Asynq Integration |
|-------------------|-------------------|
| **Auth events** (password reset, email verification) | Enqueue Asynq tasks instead of/publish events |
| **Email sending** | Asynq task: `send_email` with template + variables |
| **Report generation** | Asynq task: `generate_report` with report type + user ID |
| **KYC verification** | Asynq task: `verify_kyc` with user ID + docs |
| **Market data caching** | Asynq task: `refresh_cache` with symbol + timeframe |

## 3. Implementation Plan

### 3.1 Step 1: Add Asynq Dependency

```go
# go.mod require
github.com/hibiken/asynq v1.3.0
```

### 3.2 Step 2: Define Task Types

Create task type constants and schemas:

```go
// task/types.go
package task

const (
    TaskSendEmail    = "send_email"
    TaskGenerateReport = "generate_report"
    TaskVerifyKYC    = "verify_kyc"
    TaskRefreshCache = "refresh_cache"
)
```

### 3.2 Step 3: Define Task Payloads

```go
// task/payloads.go
type SendEmailPayload struct {
    To      string
    Template string
    Context map[string]interface{}
}

type GenerateReportPayload struct {
    UserID   string
    ReportType string
    Format   string
}
```

### 3.3 Step 4: Task Handler Functions

```go
// task/handlers.go
func HandleSendEmail(ctx context.Context, payload *SendEmailPayload) error {
    // Implement email sending logic
    // Use existing email service/SMTP/SendGrid
    return nil
}

func HandleGenerateReport(ctx context.Context, payload *GenerateReportPayload) error {
    // Implement report generation logic
    return nil
}
```

### 3.4 Step 5: Register Task Handlers

```go
// task/router.go
import (
    asynq "github.com/hibiken/asynq"
    
    "mogtrade/task"
    "mogtrade/task/handlers"
)

func RegisterTaskRouter(mux *asynq.ServeMux) {
    mux.Handle(task.TaskSendEmail, handlers.HandleSendEmail)
    mux.Handle(task.TaskGenerateReport, handlers.HandleGenerateReport)
    // Register other tasks...
}
```

### 3.4 Step 5: Create Worker Process

```go
// worker/main.go
import (
    asynq "github.com/hibiken/asynq"
    "log"
    "os"
)

func main() {
    // PostgreSQL connection
    dbURL := os.Getenv("DATABASE_URL")
    
    // Create broker (using PostgreSQL)
    broker := asynq.NewBroker(asynq.RedisBrokerOptions{
        // or use NewPostgresBroker with pgx
    })
    
    // Create worker
    worker := asynq.NewWorker(
        broker,
        asynq.WorkerConfig{
            Concurrent: 10, // number of concurrent workers
            Prefetch: 100,
        },
    )
    
    // Register task handlers
    worker.Handle(task.TaskSendEmail, handlers.HandleSendEmail)
    worker.Handle(task.TaskGenerateReport, handlers.HandleGenerateReport)
    // ...
    
    log.Println("Starting Asynq worker...")
    if err := worker.Start(); err != nil {
        log.Fatalf("failed to start worker: %v", err)
    }
}
```

### 3.4 Step 6: Update Event Bus Integration

Option A: **Replace** - Remove event bus publishing for tasks, use Asynq exclusively

Option B: **Supplement** - Keep event bus for other events, use Asynq for CPU-intensive/I/O-bound tasks

```go
// In auth controller, replace:
helpers.PublishEvent(ctx, "password_reset_requested", evt)

// With:
asynqClient := asynq.NewClient(asynq.RedisBrokerOptions{
    Address: os.Getenv("ASYNQ_ADDRESS"),
})
payload, _ := json.Marshal(SendEmailPayload{...})
task, err := asynqClient.Enqueue(task.TaskSendEmail, payload)
```

### 3.6 Step 6: Add Asynq Dashboard

```go
// in main.go or router
mux.Handle("/q", asynq.NewServeMux(
    asynq.NewBroker(options),
    asynq.NewWorker(options),
))
```

## 4. Migration Strategy

### 4.1 Phase 1: Parallel Operation (2-3 weeks)
- Keep event bus running
- Add Asynq workers in parallel
- Monitor task completion rates
- Ensure no tasks are lost

### 4.2 Phase 2: Gradual Migration (3-4 weeks)
- Migrate 50% of tasks to Asynq
- Decommission event bus handlers for migrated tasks
- Monitor system performance

### 4.3 Phase 3: Full Migration (2-3 weeks)
- Migrate remaining tasks
- Decommission event bus entirely (or keep for non-task events)
- Optimize Asynq configuration
- Remove event bus dependencies if fully replaced

### 4.2 Rollback Plan
- If Asynq fails, switch back to event bus
- Ensure idempotency of both approaches
- Maintain backwards compatibility during transition

## 5. Code Changes Checklist

### 4.1 New Files
- `task/types.go` - Task type constants
- `task/payloads.go` - Task payload structs
- `task/handlers.go` - Task handler functions
- `task/router.go` - Task router registration
- `worker/main.go` - Worker process entry point
- `asynq_config.go` - Asynq configuration

### 4.2 Modified Files
- `app/controller/auth.go` - Replace event publishing with Asynq enqueue
- `app/helper/events.go` - Remove or repurpose event publishing
- `infra/db/models.go` - Add task status tracking if needed
- `go.mod` - Add asynq dependency

### 4.3 Configuration
- `.env` additions:
  - `ASYNQ_ADDRESS` - PostgreSQL connection string or Redis address
  - `ASYNQ_CONCURRENT` - Number of worker goroutines
  - `ASYNQ_MAX_RETRIES` - Maximum retry attempts
  - `ASYNQ_TASK_TIMEOUT` - Task timeout in seconds

## 5. Monitoring & Observability

### 4.1 Asynq Dashboard
- Access at `http://localhost:8080/q`
- Monitor task enqueue rates, success/failure rates, execution times
- View failed tasks for debugging

### 4.2 Logging
- Structured logging for task start, completion, failure
- Include task type, payload hash, worker ID in log entries

### 4.2 Alerts
- Set up alerts for:
  - High task failure rates
  - Worker crash/restart frequency
  - Database connection issues
  - Timeout exceedances

## 5. Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Task loss during migration | Run parallel for period, ensure idempotency |
| Worker crashes | Implement retry with exponential backoff |
| Database pressure | Connection pooling, batch processing |
| Payload size limits | Compress large payloads, use file references |
| Existing event bus reliance | Parallel operation period, feature flags |

## 5. Performance Considerations

- **Task enqueue latency**: ~1-5ms with PostgreSQL broker
- **Worker processing time**: Depends on task type, monitor and optimize
- **Database load**: Tasks add write load to PostgreSQL, monitor pg_stat_activity
- **Memory usage**: Workers keep payloads in memory, monitor heap growth

## 6. Example: Password Reset with Asynq

### 6.1 Enqueue Task (controller)

```go
func (a *Auth) handlePasswordReset(c *gin.Context) {
    // ... validate email, find user ...
    
    // Enqueue Asynq task instead of publishing event
    payload := asynq.TaskPayload{
        "to": user.Email,
        "template": "password_reset",
        "context": map[string]interface{}{
            "name": user.Name,
            "resetURL": resetURL,
        },
    }
    
    task, err := asynqClient.Enqueue(
        task.TaskSendEmail,
        payload,
        asynq.MaxRetry(3),
        asynq.Timeout(5 * time.Minute),
    )
    if err != nil {
        // handle error
    }
    
    // Respond immediately
    c.JSON(http.StatusAccepted, gin.H{"status": "reset request enqueued"})
}
```

### 6.2 Worker Handler

```go
func HandleSendEmail(ctx context.Context, payload *SendEmailPayload) error {
    // Send email using existing SMTP/SendGrid service
    err := emailService.Send(
        To: payload.To,
        Template: payload.Template,
        Context: payload.Context,
    )
    return err
}
```

## 7. Conclusion

Asynq provides a robust, production-ready background job processing system that can significantly improve MogTrade's reliability and observability compared to a custom event bus solution. The integration plan allows for gradual migration with minimal downtime, and the dashboard provides excellent observability.

The key benefits:
- **Reliability**: PostgreSQL-backed task storage ensures no task loss
- **Observability**: Built-in dashboard for monitoring
- **Simplicity**: Easy API for enqueueing and processing tasks
- **Scalability**: Horizontal workers for load handling
- **Observability**: Built-in metrics and dashboard

## 6. Integration Complexity
- **Complexity**: Medium (3-4 weeks for full migration)
- **Expected ROI**: High (better reliability, observability, scalability)
- **Recommended For**: Projects needing reliable task processing with minimal operational overhead
- **Not Recommended For**: Projects with very simple, synchronous-only requirements

## 7. Summary

This plan provides a comprehensive guide for integrating Asynq into the MogTrade project. The integration is designed to be incremental, allowing the team to migrate at their own pace while maintaining system stability. The built-in dashboard and reliability features of Asynq make it a strong choice for background job processing in a Go microservices architecture.