# Dev MCP - Development Multi-Cloud Platform

> **🔄 Recently Refactored**: This project has been completely refactored to use the **official Model Context Protocol (MCP) Go SDK**, ensuring standards compliance, type safety, and enhanced transport support with SSE as the forced default mode.

Dev MCP is a Go-based platform that provides a unified interface for querying various data sources including databases, Grafana Loki, S3, Sentry, and Swagger APIs. It also includes a built-in HTTP request simulator for testing purposes. The platform supports both standalone mode and MCP (Model Context Protocol) mode for integration with AI assistants.

## Features

- **Database Queries**: Support for MySQL table schema inspection and data querying
- **Grafana Loki Integration**: Query logs using LogQL
- **S3 JSON Data Access**: Retrieve and parse JSON data from S3 URLs
- **Sentry Integration**: Error tracking and issue management
- **Swagger API Parsing**: Parse and inspect Swagger/OpenAPI specifications
- **Large Language Models (LLM) Service**: Support for OpenAI, Anthropic Claude, and local models
- **HTTP Request Simulation**: Simulate HTTP requests for testing
- **MCP Protocol Support**: Expose all functionality as MCP tools for AI assistant integration

## Architecture

### Official MCP Framework Integration

Dev MCP has been completely refactored to use the **official Model Context Protocol (MCP) Go SDK** (`github.com/modelcontextprotocol/go-sdk`), providing:

- **Standards Compliance**: Full compatibility with the official MCP specification
- **Type Safety**: Strongly typed tool definitions and content handling
- **Transport Flexibility**: Support for multiple transport modes (stdio, SSE, HTTP)
- **Resource Management**: Automatic discovery and registration of database tables, log streams, S3 data, and API specifications
- **Structured Logging**: Component-based logging system with debug mode support
- **Error Handling**: Comprehensive error handling with context preservation

### Key Refactoring Highlights

1. **Tool System Refactoring**: Complete rewrite using `ToolDefinition` structure with official SDK compatibility
2. **Server Implementation**: Migrated from custom implementation to official SDK `Connect` method
3. **Transport Layer**: Enhanced support for multiple transport modes with SSE as the forced default
4. **Resource Discovery**: Automatic resource management system for databases, logs, APIs, and documentation
5. **Unified Entry Point**: Streamlined main.go with mode selection and transport configuration

### MCP Protocol Implementation

The server implements the Model Context Protocol with:
- **Tools**: 7 different tool types (database, Loki, S3, Sentry, Swagger, LLM, simulator)
- **Resources**: Dynamic resource discovery for databases, logs, S3 buckets, and API specs
- **Transport**: SSE-first approach with fallback support for stdio and HTTP
- **Content Types**: Full support for text, images, and structured data through official MCP types

## Project Structure

```
dev-mcp/
├── cmd/
│   └── main.go          # Main application entry point with transport mode support
├── configs/
│   └── config.yaml      # Main configuration file
├── internal/
│   ├── config/          # Configuration loading utilities
│   ├── database/        # Database query functionality
│   ├── loki/            # Grafana Loki integration
│   ├── s3/              # S3 JSON data access
│   ├── sentry/          # Sentry integration
│   ├── swagger/         # Swagger API parsing
│   ├── llm/             # Large Language Models service
│   ├── simulator/       # HTTP request simulation
│   ├── logging/         # Structured logging system (NEW)
│   ├── errors/          # Error handling with context (NEW)
│   └── mcp/             # Official MCP protocol implementation
│       ├── server/      # MCP server with official SDK
│       ├── tools/       # Tool definitions using official SDK
│       ├── resources/   # Resource discovery and management (NEW)
│       └── types/       # MCP type definitions
├── scripts/             # Utility scripts including transport mode tests
│   ├── test-mcp.bat     # MCP functionality tests (Windows)
│   ├── test-mcp.sh      # MCP functionality tests (Unix)
│   ├── test-transport-modes.bat  # Transport mode tests (Windows) (NEW)
│   └── test-transport-modes.sh   # Transport mode tests (Unix) (NEW)
├── pkg/                 # Reusable packages
├── docs/                # Documentation
└── go.mod               # Go module definition with official MCP SDK
```

## Installation

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd dev-mcp
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

## Configuration

The application can be configured through multiple methods, with the following priority (from highest to lowest):

1. **Environment Variables** (highest priority)
2. **Configuration File** (`configs/config.yaml`)
3. **Default Values** (lowest priority)

### Environment Variables

All configuration options can be set using environment variables with the prefix `MCP_`. The variable names follow these rules:
- Convert the YAML path to uppercase
- Replace dots with underscores
- Prefix with `MCP_`

Examples:
- `server.port` becomes `MCP_SERVER_PORT`
- `database.host` becomes `MCP_DATABASE_HOST`
- `llm.providers[0].api_key` becomes `MCP_LLM_PROVIDERS_0_API_KEY`

You can also use a `.env` file to set environment variables. An example `.env.example` file is provided in the project root. To use it:

1. Copy the example file: `cp .env.example .env`
2. Modify the values as needed
3. The application will automatically load these variables

Note: Environment variables take precedence over configuration file values.

### Server Configuration

#### Configuration File
```yaml
server:
  port: 8080
  host: localhost
```

#### Environment Variables
```bash
MCP_SERVER_PORT=8080
MCP_SERVER_HOST=localhost
```

### Database Configuration

#### Configuration File
```yaml
database:
  host: localhost
  port: 3306
  username: root
  password: password
  dbname: dev_mcp
```

#### Environment Variables
```bash
MCP_DATABASE_HOST=localhost
MCP_DATABASE_PORT=3306
MCP_DATABASE_USERNAME=root
MCP_DATABASE_PASSWORD=password
MCP_DATABASE_DBNAME=dev_mcp
```

### Grafana Loki Configuration

#### Configuration File
```yaml
loki:
  host: http://localhost:3100
  username: ""
  password: ""
```

#### Environment Variables
```bash
MCP_LOKI_HOST=http://localhost:3100
MCP_LOKI_USERNAME=
MCP_LOKI_PASSWORD=
```

### S3 Configuration

#### Configuration File
```yaml
s3:
  endpoint: ""
  region: us-east-1
  access_key: ""
  secret_key: ""
  bucket: ""
```

#### Environment Variables
```bash
MCP_S3_ENDPOINT=
MCP_S3_REGION=us-east-1
MCP_S3_ACCESS_KEY=
MCP_S3_SECRET_KEY=
MCP_S3_BUCKET=
```

### Sentry Configuration

#### Configuration File
```yaml
sentry:
  dsn: ""
  environment: development
  release: "1.0.0"
```

#### Environment Variables
```bash
MCP_SENTRY_DSN=
MCP_SENTRY_ENVIRONMENT=development
MCP_SENTRY_RELEASE=1.0.0
```

### Swagger Configuration

#### Configuration File
```yaml
swagger:
  url: "/swagger/"
  filepath: "./docs/swagger.json"
```

#### Environment Variables
```bash
MCP_SWAGGER_URL=/swagger/
MCP_SWAGGER_FILEPATH=./docs/swagger.json
```

### Large Language Models (LLM) Configuration

#### Configuration File
```yaml
llm:
  providers:
    - name: "openai"
      type: "openai"
      api_key: "your-openai-api-key"
      enabled: false
      model: "gpt-3.5-turbo"
    - name: "anthropic"
      type: "anthropic"
      api_key: "your-anthropic-api-key"
      enabled: false
      model: "claude-3-haiku-20240307"
    - name: "local"
      type: "local"
      endpoint: "http://localhost:8000/v1"
      enabled: false
      model: "llama-2-7b"
```

#### Environment Variables
```bash
# OpenAI Provider
MCP_LLM_PROVIDERS_0_NAME=openai
MCP_LLM_PROVIDERS_0_TYPE=openai
MCP_LLM_PROVIDERS_0_API_KEY=your-openai-api-key
MCP_LLM_PROVIDERS_0_ENABLED=false
MCP_LLM_PROVIDERS_0_MODEL=gpt-3.5-turbo

# Anthropic Provider
MCP_LLM_PROVIDERS_1_NAME=anthropic
MCP_LLM_PROVIDERS_1_TYPE=anthropic
MCP_LLM_PROVIDERS_1_API_KEY=your-anthropic-api-key
MCP_LLM_PROVIDERS_1_ENABLED=false
MCP_LLM_PROVIDERS_1_MODEL=claude-3-haiku-20240307

# Local Provider
MCP_LLM_PROVIDERS_2_NAME=local
MCP_LLM_PROVIDERS_2_TYPE=local
MCP_LLM_PROVIDERS_2_ENDPOINT=http://localhost:8000/v1
MCP_LLM_PROVIDERS_2_ENABLED=false
MCP_LLM_PROVIDERS_2_MODEL=llama-2-7b
```

## Usage

### Standalone Mode
Run the application in standalone mode for basic development and testing:
```bash
go run cmd/main.go
```

This will perform health checks and show available services.

### MCP Server Mode
To start the application in MCP (Model Context Protocol) mode with transport support:

#### Basic MCP Mode (Default SSE Transport)
```bash
go run cmd/main.go mcp
```

#### Explicit Transport Mode Selection
```bash
# SSE (Server-Sent Events) mode - Recommended
go run cmd/main.go mcp --sse

# HTTP mode (uses SSE implementation)
go run cmd/main.go mcp --http

# Using transport parameter
go run cmd/main.go mcp --transport sse
```

#### Debug Mode
```bash
# Enable debug logging
go run cmd/main.go mcp --sse --debug
```

### Transport Modes

Dev MCP supports three transport modes for MCP communication:

1. **SSE (Server-Sent Events)** - **DEFAULT/FORCED MODE**
   - HTTP-based transport using Server-Sent Events
   - Best for web-based AI assistants
   - Provides real-time bidirectional communication
   - Default port: 8080

2. **HTTP** - Uses SSE implementation
   - Standard HTTP transport with SSE backend
   - Compatible with web browsers and HTTP clients

3. **stdio** - **Forced to SSE Mode**
   - Traditional stdio communication (forced to SSE for better compatibility)
   - All stdio requests are automatically redirected to SSE mode

**Note**: As requested, all transport modes currently force the use of SSE (Server-Sent Events) transport for optimal compatibility and performance.

### Available Commands

| Command | Description |
|---------|-------------|
| `go run cmd/main.go` | Run in standalone mode with health checks |
| `go run cmd/main.go mcp` | Start MCP server with default SSE transport |
| `go run cmd/main.go mcp --sse` | Start MCP server with explicit SSE transport |
| `go run cmd/main.go mcp --http` | Start MCP server with HTTP transport (uses SSE) |
| `go run cmd/main.go mcp --stdio` | Start MCP server (forced to SSE mode) |
| `go run cmd/main.go mcp --debug` | Start MCP server with debug logging |

#### Available MCP Tools (Official SDK Implementation)

Dev MCP exposes 7 powerful tools through the official MCP SDK:

1. **`database_query`** - Database Operations
   - Query MySQL database tables and schemas
   - Execute SQL SELECT statements safely
   - Retrieve table structures and metadata
   - Support for parameterized queries

2. **`loki_query`** - Log Analysis
   - Query Grafana Loki logs using LogQL syntax
   - Time-range based log retrieval
   - Label filtering and aggregation
   - Stream-based log data access

3. **`s3_query`** - Cloud Storage Access
   - Retrieve and parse JSON data from S3 URLs
   - Support for authenticated S3 buckets
   - Automatic JSON parsing and formatting
   - Cross-region S3 access

4. **`sentry_query`** - Error Tracking
   - Query Sentry issues and error events
   - Filter by project, environment, and time
   - Retrieve error details and stack traces
   - Monitor application health metrics

5. **`swagger_query`** - API Documentation
   - Parse Swagger/OpenAPI specifications
   - Extract endpoint definitions and schemas
   - Validate API documentation structure
   - Generate API usage examples

6. **`llm_chat`** - AI Integration
   - Interact with multiple LLM providers (OpenAI, Anthropic, Local)
   - Support for chat completions and text generation
   - Provider-agnostic interface
   - Model configuration and selection

7. **`http_request`** - Request Simulation
   - Simulate HTTP requests for testing
   - Support for all HTTP methods (GET, POST, PUT, DELETE, etc.)
   - Custom headers and payload configuration
   - Response parsing and validation

#### MCP Resources (Auto-Discovery)

The server automatically discovers and exposes resources through the official MCP SDK:

- **Database Tables**: Automatically discovered table schemas and metadata
- **Log Streams**: Available Loki log streams and labels
- **S3 Objects**: Accessible S3 bucket contents and JSON data
- **API Specifications**: Available Swagger/OpenAPI documentation
- **Error Reports**: Sentry project issues and error summaries

#### Testing MCP Functionality

**Basic MCP Tests:**
- On Unix/Linux/macOS: `scripts/test-mcp.sh`
- On Windows: `scripts/test-mcp.bat`

**Transport Mode Tests:**
- On Unix/Linux/macOS: `scripts/test-transport-modes.sh`
- On Windows: `scripts/test-transport-modes.bat`

**Manual Testing Examples:**
```bash
# Test SSE transport mode (default)
go run cmd/main.go mcp --sse --debug

# Test HTTP transport mode (uses SSE backend)
go run cmd/main.go mcp --http

# Test with transport parameter
go run cmd/main.go mcp --transport sse

# Test stdio mode (forced to SSE)
go run cmd/main.go mcp --stdio
```

**Build and Test:**
```bash
# Build the project
go build -o dev-mcp cmd/main.go

# Run tests
go test ./...

# Test with different transport modes
./dev-mcp mcp --sse    # SSE mode
./dev-mcp mcp --http   # HTTP mode (uses SSE)
./dev-mcp mcp --stdio  # stdio mode (forced to SSE)
```

## Large Language Models (LLM) Service

The LLM service provides a unified interface to work with different large language models:

### Supported Providers
- **OpenAI**: GPT-3.5, GPT-4, and other OpenAI models
- **Anthropic**: Claude models
- **Local Models**: Any model with an OpenAI-compatible API

### Configuration
To enable LLM providers, set `enabled: true` in the configuration and provide the required API keys or endpoints.

### Example Usage
```go
// Chat with a model
chatReq := &models.ChatRequest{
    Model: "gpt-3.5-turbo",
    Messages: []models.Message{
        {Role: models.UserRole, Content: "Hello, how are you?"},
    },
}

chatResp, err := llmService.Chat(context.Background(), chatReq)
if err != nil {
    log.Fatal(err)
}

fmt.Println(chatResp.Choices[0].Message.Content)
```

## API Endpoints

(TODO: Define API endpoints for each service)

## Development

### Adding New Features

1. Create a new package in `internal/` for the feature
2. Implement the functionality
3. Add configuration options to `configs/config.yaml`
4. Initialize the feature in `cmd/main.go`

### Testing

Run unit tests with:
```bash
go test ./...
```

## Dependencies

### Core Dependencies
- `github.com/modelcontextprotocol/go-sdk` - **Official MCP Go SDK** (NEW)
- `github.com/go-sql-driver/mysql` - MySQL driver
- `github.com/go-resty/resty/v2` - REST client for HTTP requests
- `github.com/aws/aws-sdk-go` - AWS SDK for S3 integration
- `github.com/getsentry/sentry-go` - Sentry SDK for error tracking
- `gopkg.in/yaml.v2` - YAML configuration parsing

### Legacy Dependencies
- `github.com/lib/pq` - PostgreSQL driver (deprecated, kept for compatibility)

### Key Features of Official MCP SDK
- Standards-compliant Model Context Protocol implementation
- Type-safe tool and resource definitions
- Multiple transport support (stdio, SSE, HTTP)
- Structured content handling with official MCP types
- Session management and lifecycle handling

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

(TODO: Add license information)