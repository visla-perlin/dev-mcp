# Dev MCP - Development Multi-Cloud Platform

---

## README.md

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

### Available MCP Tools

All providers implement the `ProviderClient` interface with `Test(config)` and `AddTools(server, config)` methods.

#### Sentry Provider
- **sentry_get_issues**: Get Sentry issues with optional filtering
  - Parameters: `query` (string, optional), `limit` (integer, default: 50)
- **sentry_get_issue_details**: Get detailed information about a specific Sentry issue
  - Parameters: `issue_id` (string, required)
- **sentry_create_issue**: Create a new Sentry issue for testing purposes
  - Parameters: `title` (string, required), `message` (string, required), `level` (string, default: "error")

#### Loki Provider
- **loki_query**: Query Grafana Loki logs using LogQL
  - Parameters: `query` (string, required), `limit` (integer, default: 100)
- **loki_labels**: Get available log labels from Loki
  - Parameters: None

#### Database Provider
- **database_query**: Execute SQL queries with security validation
  - Parameters: `query` (string, required)
- **database_schema**: Get table schema information
  - Parameters: `table` (string, optional)

#### S3 Provider
- **s3_get_object**: Retrieve objects from S3
  - Parameters: `bucket` (string, required), `key` (string, required)
- **s3_list_objects**: List objects in S3 bucket
  - Parameters: `bucket` (string, required), `prefix` (string, optional), `limit` (integer, default: 100)

#### File Provider
- **file_read**: Read file contents with security validation
  - Parameters: `path` (string, required)
- **file_list**: List files in directory with security validation
  - Parameters: `path` (string, default: "."), `pattern` (string, optional)

### Provider Architecture

Each provider follows the same pattern:
1. **Test Configuration**: `Test(config interface{}) error` - validates and tests the configuration
2. **Add Tools**: `AddTools(server *mcp.Server, config interface{}) error` - registers tools if test passes
3. **Internal Implementation**: Uses existing service clients to provide functionality
4. **Security**: Built-in security validation for dangerous operations

## Project Structure

...existing code...

---

## README_REFACTORED.md

# Dev MCP Server

A comprehensive Model Context Protocol (MCP) server implementation using the official Go MCP SDK, providing unified access to various data sources and services.

...existing code...

---

## CONNECTION_GUIDE.md

# Dev MCP - 连接配置说明

...existing code...

---

## CONFIGURATION_GUIDE.md

# MCP SSE Configuration Guide

...existing code...

---

## ARCHITECTURE.md

# Dev MCP 架构优化总结

...existing code...

---

*本文件由自动合并脚本生成，包含所有 Markdown 文档内容。*