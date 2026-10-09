# background-jobs.md

## Background Jobs & Scheduling

The MogTrade Engine uses a job scheduling system to handle background tasks and periodic operations.

### Job System Architecture

- **Core Abstraction**: Defined in `core/contract/jobs.go` with `Job` interface
- **Scheduler**: `JobScheduler` interface for registering and managing jobs
- **Execution**: Jobs run in separate goroutines with context awareness
- **Error Handling**: Jobs can retry or fail gracefully

### Job Types

1. **Stale Token Remover Job** (`services/jobs/refresh-token-cleaner.go`)
   - Scheduled: Every Sunday at midnight (0 0 * * 0)
   - Purpose: Clean up expired refresh tokens
   - Dependencies: Database queries, job scheduler

2. **Other Job Types**
   - Can be implemented following same pattern
   - Requires `Name()`, `Schedule()`, and `Run()` methods
   - Uses context for cancellation and timeouts

### Scheduling Mechanism

- **Registration**: Jobs registered in `setupBackgroundJobs()` (ioc.go)
- **Timing**: Cron-like scheduling using standard cron expressions
- **Execution Context**: Jobs receive context with timeout handling
- **Logging**: Jobs log their activities for observability

### Implementation Pattern

```go
type Job interface {
    Name() string
    Schedule() any  // Returns scheduling spec or channel
    Run(ctx context.Context) error
}

func (j *MyJob) Schedule() any {
    return "0 0 * * 0"  // Every Sunday at midnight
}

func (j *MyJob) Run(ctx context.Context) error {
    // Job implementation
    return nil
}
```

### Job Lifecycle

1. **Registration**: Jobs added to scheduler during app startup
2. **Execution**: Scheduler launches jobs with context
3. **Completion**: Jobs signal completion via context cancellation
4. **Monitoring**: Job status tracked and logged

### Key Considerations

- **Idempotency**: Jobs should be able to run multiple times safely
- **State Management**: Jobs may need to track state between runs
- **Resource Usage**: Background jobs should be efficient
- **Error Handling**: Jobs should handle failures gracefully
- **Testing**: Jobs should be testable in isolation