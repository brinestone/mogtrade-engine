# MogTrade - Trading Platform

## Overview

MogTrade is a comprehensive trading platform engine that provides a RESTful API for frontends and clients to connect to and execute trades on connected stock market exchanges. The platform is designed with high performance and low latency in mind, featuring a modular architecture that separates concerns across four main modules: app, core, infra, and services.

## Project Structure

### app
- Contains the main entrypoint executable
- Uses GIN framework for REST API hosting
- Implements IoC Container for dependency injection
- Contains API controllers, middleware, and HTTP handlers

### core
- Contains core library interfaces that can be used throughout the project
- Defines interfaces for data access, authentication, and other core concepts

### infra
- Contains implementation details for external system integrations
- Handles communication with external systems like stock exchanges
- Contains database integration code and migration utilities

### services
- Contains service functions for business logic and core trading functionality
- Implements the core library interfaces defined in the core module

### go.work and go.sum
- Link the different modules together
- Manage Go module dependencies

## Key Features

- RESTful API for trading, account management, and market data
- Support for multiple exchange data feeds (currently Massive API)
- Secure authentication with JWT and multi-factor authentication

<tool_call>
<function=write>
<parameter=content>
# MogTrade - Trading Platform

## Overview

MogTrade is a comprehensive trading platform engine that provides a RESTful API for frontends and clients to connect to and execute trades on connected stock market exchanges. The platform is designed with high performance and low latency in mind, featuring a modular architecture that separates concerns across four main modules: app, core, infra, and services.

## Project Structure

### app
- Contains the main entrypoint executable
- Uses GIN framework for REST API hosting
- Implements IoC Container for dependency injection
- Contains API controllers, middleware, and HTTP handlers

### core
- Contains core library interfaces that can be used throughout the project
- Defines interfaces for data access, authentication, and other core concepts

### infra
- Contains implementation details for external system integrations
- Handles communication with external systems
- Contains database integration code and migration utilities

### services
- Contains service functions for business logic and core trading functionality
- Implements the core library interfaces defined in the core module

### go.work and go.sum
- Link the different modules together
- Manage Go module dependencies

## Key Features

- RESTful API for trading, account management, and market data
- Support for multiple exchange data feeds (currently Massive API)
- Secure authentication with JWT and multi-factor authentication
- Rate limiting and abuse protection mechanisms
- Session management with JWT tokens
- Comprehensive logging and error tracking via Sentry
- Monitoring and observability with Prometheus and Grafana
- SQLC-generated domain models and query functions
- IoC Container for dependency injection

## Massive API Integration

The MassiveDatasource implements the feed.Datasource interface to fetch aggregate bar data from the Massive API (formerly Polygon.io). Key features include:

- Fetches recent aggregate bars for a given symbol and time interval
- Converts Massive's response format to the internal TradeData structure
- Implements the feed.Datasource interface seamlessly
- Configurable request timeout via MassiveConfig
- Handles API rate limits and errors gracefully

### API Endpoint
```
https://api.polygon.io/v2/aggs/ticker/{symbol}/range/{multiplier}/{timespan}/{from}/{to}?apiKey={apikey}
```

**Supported parameters:**
- `symbol`: Trading symbol (e.g., "AAPL")
- `interval`: Time range interval (5min, 30min, 60min, daily, weekly, monthly)
- `from`/`to`: Date range in YYYY-MM-DD format

**Response fields converted to TradeData:**
- Open, High, Low, Close prices
- Volume

## Configuration

### Environment Variables

The following environment variable is required (already present in `app/cmd/.env`):

```python
MASSIVE_API_KEY=k2ZDg6YupZ45k7Wh8DtMpvplzwvm1cuD
```

### IOC Configuration

The integration is configured in `app/cmd/ioc.go`:
```go
if err := ioc.Factory(func(h *market.ExchangeHub, l *slog.Logger) *market.ExchangePoller {
    return market.NewExchangePoller(l.With("service", "exchange-poller"), h, []feed.Datasource{
        feed.NewMassiveDatasource(os.Getenv("MASSIVE_API_KEY"), feed.MassiveConfig{RequestTimeout: 10 * time.Second}),
    })
}, true); err != nil {
    return err
}
```

## Building and Running

The project builds successfully with:
```bash
cd I:/mogtrade-engine/app && go build ./...
```

## Development Tools

- **air**: For running local development application (description in .air.toml)
- **mcp server**: Use the `mogtrade-dev-mcp` MCP server for database migration file generation
- **sqlc**: For generating domain data models and query functions

## Contribution Guidelines

### How to Contribute

1. **Fork the repository** and create a feature branch:
   ```
   git checkout -b feature/your-feature
   ```

2. **Follow coding standards**:
   - Use `gofmt` to format code
   - Follow Go conventions (error handling, naming, etc.)
   - Use `new(any)` built-in function to obtain pointers to values
   - Avoid using `interface{}` type, use `any` type where needed

3. **Testing**:
   - Write unit tests for all new functionality
   - Ensure existing tests continue to pass
   - Run `go test ./...` regularly

4. **Pull Request Process**:
   - Submit PR with clear description of changes
   - Include relevant screenshots or logs if needed
   - Address reviewer comments promptly

### Contribution Process

1. Fork the repository
2. Create a new branch with descriptive name (e.g., `feat/auth-jwt`)
3. Make your changes
4. Commit with clear, descriptive messages
5. Push to your fork
6. Open a Pull Request

### Coding Standards

- Use `gofmt` for code formatting
- Follow Go conventions for naming and structure
- Avoid using `interface{}` type, use `any` type where needed
- Use `new(any)` built-in function to obtain pointers to values
- Ignore `.vfox` folder and `app/cmd/.env` file (contains secret values)
- Ignore all files listed in `.gitignore` files

### Testing

- Write unit tests for all new functionality
- Ensure existing tests continue to pass
- Run `go test ./...` before pushing changes
- Use `go test -cover` to check coverage

### Code Review

- PRs will be reviewed by project maintainers
- Address feedback promptly
- Do not merge without approval

## Testing

- Run `go test ./...` to execute all tests
- Use `go test -cover` to check coverage
- Ensure all tests pass before submitting PR

## License

MIT License

## Contact

For questions or support, please contact the project maintainers via GitHub Issues or email.