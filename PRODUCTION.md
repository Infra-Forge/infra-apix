# Production Usage Guide

This guide covers best practices for using the `apix` library in production applications.

## Table of Contents
- [Installation](#installation)
- [Production Configuration](#production-configuration)
- [Security Best Practices](#security-best-practices)
- [Performance Optimization](#performance-optimization)
- [Best Practices](#best-practices)

## Installation

### Adding to Your Project

```bash
# Install the latest version
go get github.com/Infra-Forge/apix

# Or install a specific version
go get github.com/Infra-Forge/apix@v1.0.0
```

### go.mod

```go
module github.com/yourorg/yourapp

go 1.21

require (
    github.com/Infra-Forge/apix v1.0.0
    github.com/labstack/echo/v4 v4.11.0  // or your framework of choice
)
```

## Production Configuration

### 1. Environment-Based Configuration

```go
package main

import (
    "os"
    "time"
    "github.com/Infra-Forge/apix/runtime"
)

func newProductionHandler() (*runtime.Handler, error) {
    env := os.Getenv("ENV")
    
    cfg := runtime.Config{
        Title:           "Your API",
        Version:         os.Getenv("API_VERSION"),
        Format:          "json",
        Validate:        true,
        CacheTTL:        5 * time.Minute, // Cache spec for 5 minutes
        EnableSwaggerUI: env != "production", // Disable Swagger UI in prod
    }
    
    // Production servers
    if env == "production" {
        cfg.Servers = []string{"https://api.yourcompany.com"}
    } else if env == "staging" {
        cfg.Servers = []string{"https://staging-api.yourcompany.com"}
    } else {
        cfg.Servers = []string{"http://localhost:8080"}
    }
    
    return runtime.NewHandler(cfg)
}
```

### 2. Conditional Swagger UI

**Option A: Disable in Production**
```go
cfg := runtime.Config{
    EnableSwaggerUI: os.Getenv("ENV") != "production",
}
```

**Option B: Protect with Authentication**
```go
// Wrap Swagger UI with auth middleware
func protectedSwaggerUI(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Check API key or JWT
        apiKey := r.Header.Get("X-API-Key")
        if apiKey != os.Getenv("SWAGGER_API_KEY") {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}

// Apply middleware
mux.Handle("/swagger", protectedSwaggerUI(http.HandlerFunc(handler.swaggerUI)))
```

### 3. Spec Caching Strategy

```go
cfg := runtime.Config{
    // Development: No caching (always fresh)
    CacheTTL: 0,
    
    // Staging: Short cache (1 minute)
    CacheTTL: 1 * time.Minute,
    
    // Production: Long cache (10 minutes)
    CacheTTL: 10 * time.Minute,
}
```

## Security Best Practices

### 1. API Security Schemes

```go
cfg.CustomizeBuilder = func(b *openapi.Builder) {
    b.SecuritySchemes = openapi3.SecuritySchemes{
        "BearerAuth": &openapi3.SecuritySchemeRef{
            Value: openapi3.NewJWTSecurityScheme(),
        },
        "ApiKeyAuth": &openapi3.SecuritySchemeRef{
            Value: &openapi3.SecurityScheme{
                Type: "apiKey",
                In:   "header",
                Name: "X-API-Key",
            },
        },
    }
    
    // Apply global security
    b.GlobalSecurity = []openapi3.SecurityRequirement{
        {"BearerAuth": []string{}},
    }
}
```

### 2. Rate Limiting

```go
import "golang.org/x/time/rate"

var limiter = rate.NewLimiter(10, 20) // 10 req/sec, burst of 20

func rateLimitMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !limiter.Allow() {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}

// Apply to OpenAPI endpoints
mux.Handle("/openapi.json", rateLimitMiddleware(handler))
```

### 3. CORS Configuration

```go
// Echo
e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
    AllowOrigins: []string{"https://yourapp.com"},
    AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
}))

// Chi
r.Use(cors.Handler(cors.Options{
    AllowedOrigins:   []string{"https://yourapp.com"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
    AllowCredentials: true,
}))
```

## Performance Optimization

### 1. Spec Caching

The runtime handler includes built-in caching:

```go
cfg := runtime.Config{
    CacheTTL: 10 * time.Minute, // Regenerate spec every 10 minutes
}
```

### 2. Compression

```go
// Echo
e.Use(middleware.Gzip())

// Chi
r.Use(middleware.Compress(5))

// Standard library
import "github.com/NYTimes/gziphandler"
mux.Handle("/openapi.json", gziphandler.GzipHandler(handler))
```

### 3. CDN for Swagger UI Assets

The current implementation uses unpkg.com CDN for Swagger UI assets, which provides:
- Global edge caching
- Automatic HTTPS
- High availability
- No bandwidth costs

For air-gapped environments, see [Self-Hosted Swagger UI](#self-hosted-swagger-ui) below.

## Integration Examples

### Echo Framework

```go
import (
    "github.com/Infra-Forge/apix"
    echoadapter "github.com/Infra-Forge/apix/echo"
    "github.com/Infra-Forge/apix/runtime"
    "github.com/labstack/echo/v4"
)

func main() {
    e := echo.New()
    adapter := echoadapter.New(e)

    // Register routes
    echoadapter.Post(adapter, "/api/items", createItem,
        apix.WithSummary("Create item"),
        apix.WithTags("items"),
    )

    // Serve OpenAPI
    handler, _ := runtime.NewHandler(runtime.Config{
        Title:           "My API",
        Version:         "1.0.0",
        EnableSwaggerUI: os.Getenv("ENV") != "production",
        CacheTTL:        5 * time.Minute,
    })
    handler.RegisterEcho(e)

    e.Start(":8080")
}
```

### Chi Framework

```go
import (
    chiadapter "github.com/Infra-Forge/apix/chi"
    "github.com/go-chi/chi/v5"
)

func main() {
    r := chi.NewRouter()
    adapter := chiadapter.New(r)

    chiadapter.Get(adapter, "/api/items/{id}", getItem,
        apix.WithSummary("Get item"),
    )

    // Serve OpenAPI
    handler, _ := runtime.NewHandler(runtime.Config{...})
    mux := http.NewServeMux()
    handler.RegisterHTTP(mux)
    r.Mount("/", mux)

    http.ListenAndServe(":8080", r)
}
```

## Advanced Features

### Custom Swagger UI

For air-gapped environments, see `SELF_HOSTED_SWAGGER.md` for embedding Swagger UI assets.

## Best Practices Summary

✅ **DO:**
- Use semantic versioning for releases
- Enable spec caching in production (`CacheTTL: 10 * time.Minute`)
- Disable or protect Swagger UI in production
- Use environment variables for configuration
- Implement health checks for orchestrators
- Use HTTPS in production
- Apply rate limiting to public endpoints
- Compress responses (gzip)
- Monitor with metrics and logging

❌ **DON'T:**
- Expose Swagger UI publicly without authentication
- Disable spec validation in production
- Hard-code secrets or API keys
- Skip CORS configuration
- Deploy without health checks
- Use development settings in production

## Example Production Setup

See `examples/production/` for a complete production-ready example with:
- Environment-based configuration
- Docker & Kubernetes manifests
- Health checks & metrics
- Security middleware
- Logging & monitoring

