# Examples

This directory contains reference examples showing how to use the `apix` library with different Go web frameworks.

## Available Examples

### Basic Examples
- **`echo/`** - Minimal Echo framework example
- **`production/`** - Production-ready configuration example

### Complete Examples (InfraNotes)
Full-featured financial analytics API demonstrating:
- Nested struct schemas (DocumentListResponse, TransactionListResponse)
- Complex types (UUID, decimal.Decimal, time.Time)
- Full CRUD operations
- Pagination
- Security schemes
- OpenAPI 3.1.0 generation

Framework-specific implementations:
- **`infranotes/`** - Echo framework
- **`infranotes-chi/`** - Chi router
- **`infranotes-gin/`** - Gin framework
- **`infranotes-mux/`** - Gorilla Mux

## Running Examples

These are **reference examples** meant to be copied to your own project. To try them:

```bash
# Navigate to an example
cd examples/infranotes

# Run directly
go run main.go

# Or build and run
go build -o myapp
./myapp
```

The server will start and expose:
- OpenAPI spec at `/openapi.json`
- Swagger UI at `/swagger`
- Health check at `/health`

## Using in Your Project

1. **Install the library:**
```bash
go get github.com/Infra-Forge/apix
```

2. **Copy relevant code** from the examples

3. **Adapt to your needs**

## Example Structure

Each example demonstrates:
- ✅ Route registration with type-safe handlers
- ✅ Request/response models
- ✅ OpenAPI metadata (summaries, descriptions, tags)
- ✅ Security schemes
- ✅ Runtime OpenAPI spec generation
- ✅ Swagger UI integration

## Notes

- Examples use `replace` directive in `go.mod` to reference the local library
- In your project, use the published version: `github.com/Infra-Forge/apix v1.0.0`
- See `../PRODUCTION.md` for production deployment guidance

