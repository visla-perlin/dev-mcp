# Dev MCP Server

A comprehensive Model Context Protocol (MCP) server implementation using the official Go MCP SDK, providing unified access to various data sources and services.

## 🚀 Features

### Tools
- **Database Query Tool**: Execute SQL queries and retrieve table schemas
- **Loki Query Tool**: Query Grafana Loki logs using LogQL
- **S3 Query Tool**: Retrieve and parse JSON data from S3 buckets
- **Sentry Query Tool**: Access Sentry issues and error tracking data
- **Swagger Query Tool**: Parse and query OpenAPI/Swagger specifications
- **LLM Tool**: Interact with large language models for chat and completion
- **HTTP Simulator Tool**: Simulate HTTP requests for testing purposes

### Resources
- **Database Tables**: Automatic discovery of database tables with schema information
- **Log Streams**: Loki log streams discovery and metadata
- **S3 Data Access**: Resource-based access to S3 JSON data
- **API Specifications**: Swagger/OpenAPI specification resources

### Enhanced Features
- **Structured Logging**: Component-based logging with configurable levels
- **Error Handling**: Comprehensive error tracking with context and stack traces
- **Resource Discovery**: Automatic resource discovery and registration
- **Health Checks**: Built-in health monitoring for all services

## 🏗️ Architecture

The server is built using the official MCP SDK and follows a modular architecture:

```
dev-mcp/
├── cmd/
│   └── main.go                 # Application entry point
├── internal/
│   ├── config/                 # Configuration management
│   ├── database/               # Database connectivity
│   ├── errors/                 # Structured error handling
│   ├── llm/                    # LLM service integration
│   ├── logging/                # Structured logging
│   ├── loki/                   # Grafana Loki integration
│   ├── mcp/
│   │   ├── resources/          # MCP resource managers
│   │   ├── server/             # MCP server implementation
│   │   └── tools/              # MCP tool implementations
│   ├── s3/                     # AWS S3 integration
│   ├── sentry/                 # Sentry error tracking
│   ├── simulator/              # HTTP request simulation
│   └── swagger/                # OpenAPI/Swagger parsing
└── configs/
    └── config.yaml             # Configuration file
```

## 🛠️ Usage

### Development Mode
```bash
# Run in development/demo mode
go run cmd/main.go

# Enable debug logging
go run cmd/main.go --debug
```

### MCP Server Mode
```bash
# Start as MCP server
go run cmd/main.go mcp

# Start with debug logging
go run cmd/main.go --debug mcp
```

### Configuration

Configure the server by editing `configs/config.yaml`:

```yaml
server:
  host: "localhost"
  port: 8080

database:
  host: "localhost"
  port: 3306
  username: "user"
  password: "password"
  dbname: "database"

loki:
  url: "http://localhost:3100"

s3:
  region: "us-east-1"
  bucket: "my-bucket"

# ... other service configurations
```

## 🔧 Tools Usage

### Database Query
```json
{
  "name": "database_query",
  "arguments": {
    "query": "SELECT * FROM users LIMIT 10"
  }
}
```

### Loki Query
```json
{
  "name": "loki_query",
  "arguments": {
    "query": "{job=\"api\"} |= \"error\"",
    "limit": 100
  }
}
```

### S3 Query
```json
{
  "name": "s3_query",
  "arguments": {
    "url": "s3://my-bucket/data/file.json"
  }
}
```

### LLM Chat
```json
{
  "name": "llm_chat",
  "arguments": {
    "model": "gpt-3.5-turbo",
    "messages": [
      {"role": "user", "content": "Hello, how are you?"}
    ]
  }
}
```

### HTTP Request Simulation
```json
{
  "name": "http_request",
  "arguments": {
    "method": "GET",
    "url": "https://api.example.com/data",
    "headers": {
      "Authorization": "Bearer token"
    }
  }
}
```

## 📚 Resources

The server automatically discovers and exposes resources:

- `database://tables/{table_name}` - Database table schemas and sample data
- `loki://streams/{label}` - Log stream metadata and query hints
- `s3://buckets/data` - S3 data access information
- `swagger://api/specification` - Complete API specifications
- `swagger://api/paths` - API paths and operations

## 🔍 Logging

The server uses structured logging with multiple levels:

- `DEBUG`: Detailed debugging information
- `INFO`: General operational messages
- `WARN`: Warning conditions
- `ERROR`: Error conditions
- `FATAL`: Fatal errors that cause shutdown

Enable debug mode with `--debug` flag for verbose logging.

## 🚨 Error Handling

Comprehensive error handling with:

- Component-specific error tracking
- Error context and stack traces
- Error chaining and unwrapping
- Structured error reporting

## 🧩 Recent Refactoring

The codebase has been completely refactored to use the official MCP SDK:

### ✅ Completed
1. **Tool Interface Refactoring**: Updated tool interfaces to be fully compatible with the official MCP SDK
2. **Server Implementation**: Optimized MCP server implementation using official SDK patterns
3. **Unified Entry Point**: Streamlined main.go to use only the official MCP server
4. **Resource Management**: Added comprehensive resource discovery and management
5. **Enhanced Logging & Error Handling**: Implemented structured logging and error tracking

### Key Improvements
- Full compatibility with official MCP SDK v1.1.0
- Automatic resource discovery and registration
- Structured logging with component-based organization
- Comprehensive error handling with context preservation
- Clean separation of concerns between tools and resources
- Support for debug mode and development testing

## 🔗 Dependencies

- [Official Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk) v1.1.0
- MySQL Driver for database connectivity
- AWS SDK for S3 integration
- Sentry SDK for error tracking
- Additional service-specific dependencies

## 📖 Development

To extend the server:

1. **Add New Tools**: Implement tools in `internal/mcp/tools/`
2. **Add New Resources**: Implement resource managers in `internal/mcp/resources/`
3. **Add New Services**: Create service clients in `internal/`
4. **Update Configuration**: Extend `internal/config/` for new settings

The architecture is designed to be modular and extensible while maintaining compatibility with the official MCP specification.