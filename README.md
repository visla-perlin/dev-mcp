# Dev MCP - Development Multi-Cloud Platform

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

## Project Structure

```
dev-mcp/
├── cmd/
│   └── main.go          # Main application entry point
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
│   └── mcp/             # MCP protocol implementation
├── pkg/                 # Reusable packages
├── docs/                # Documentation
├── scripts/             # Utility scripts
└── go.mod               # Go module definition
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

### Normal Mode
1. Start the application:
   ```bash
   go run cmd/main.go
   ```

2. The application will initialize all clients and display status information.

### MCP Mode
To start the application in MCP (Model Context Protocol) mode:
```bash
go run cmd/main.go mcp
```

In MCP mode, the application will expose all its functionality as tools that can be accessed through the Model Context Protocol interface.

#### Available MCP Tools
- `database_query` - Query database tables and retrieve data
- `loki_query` - Query Grafana Loki logs using LogQL
- `s3_query` - Retrieve and parse JSON data from S3 URLs
- `sentry_query` - Query Sentry issues and errors
- `swagger_query` - Parse and query Swagger/OpenAPI specifications
- `llm_chat` - Interact with large language models for chat and text generation
- `http_request` - Simulate HTTP requests for testing

#### Testing MCP Functionality
You can test the MCP functionality using the provided test scripts:
- On Unix/Linux/macOS: `scripts/test-mcp.sh`
- On Windows: `scripts/test-mcp.bat`

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

- `github.com/lib/pq` - PostgreSQL driver (deprecated, but kept for compatibility)
- `github.com/go-sql-driver/mysql` - MySQL driver
- `github.com/go-resty/resty/v2` - REST client
- `github.com/aws/aws-sdk-go` - AWS SDK for S3 integration
- `github.com/getsentry/sentry-go` - Sentry SDK
- `gopkg.in/yaml.v2` - YAML parsing

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

(TODO: Add license information)