# infrastructure-integrations.md

## Infrastructure & External Integrations

The infrastructure layer provides concrete implementations of core abstractions and integrates with external systems.

### Database Layer

- **SQLC-Generated Models**: Located in `infra/db/`, generated from SQL schemas
- **Repository Pattern**: Uses `db.Queries` interface for data access
- **Connection Management**: pgx connection pooling with transaction support
- **Transaction Support**: All critical operations use transactions for data integrity

### Event Bus

- **In-Memory Bus**: `infra/events/bus.go` provides simple publish-subscribe mechanism
- **Thread-Safe**: Concurrent access supported
- **Context-Aware**: Operations respect request context
- **Lightweight**: Minimal overhead for internal communication

### Mail Service

- **Mailtrap Integration**: Uses Mailtrap API for email sending
- **SMTP Configuration**: Configurable host, port, authentication
- **Template System**: Supports HTML and text email templates
- **Error Handling**: Retries and dead-letter queue mechanisms

### Identity & Authentication

- **UUID Generation**: ULID-based ID generation for orders, users, etc.
- **JWT Tokens**: Signed tokens with claims validation
- **Token Storage**: Secure storage of refresh tokens
- **Revocation**: Mechanisms for token invalidation

### External Services

- **Market Data**: External market data providers (Massive, AlphaVantage)
- **Email**: Mailtrap for development, other SMTP providers for production
- **Secrets Management**: Environment variables for credentials
- **Monitoring**: Integration with logging and monitoring systems

### Security Considerations

- **Input Validation**: All external inputs validated
- **Rate Limiting**: Prevents abuse of external APIs
- **Encryption**: Sensitive data encrypted at rest and in transit
- **Audit Logging**: Tracks access to sensitive operations

### Compliance

- **PCI-DSS**: Order data handling follows PCI standards
- **GDPR**: User data management includes deletion mechanisms
- **Audit Trails**: Complete logs of critical operations