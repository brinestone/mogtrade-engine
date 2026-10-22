# Build stage for the MogTrade Web API module
FROM golang:1.26-alpine AS builder

# Set working directory
WORKDIR /workspace

# Copy go module files
COPY app/go.mod app/go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY app/ ./app/


# Build the binary
# The entry point is app/cmd/main.go which uses godotenv and ioc container
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o mogtrade ./app/cmd/main.go

# Runtime stage
FROM alpine:3.20

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Set working directory
WORKDIR /home/appuser

# Copy the binary from builder
COPY --from=builder /workspace/mogtrade ./

# Expose the API port
EXPOSE 8080

# Environment variables (can be overridden at container runtime)
# - PORT: server port (default 8080)
# - LOGGING: enable/disable file logging (default disable)
# - DB_URL: database connection string
# - ALLOWED_ORIGINS: CORS allowed origins (semicolon-separated)
ENV PORT=8080 \
    LOGGING=disable

# Run the API server
# The app expects .env file or environment variables to be set
CMD ["/mogtrade"]
