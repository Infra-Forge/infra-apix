# Self-Hosted Swagger UI Guide

This guide shows how to embed Swagger UI assets directly in your binary for air-gapped or high-security environments.

## Why Self-Host?

**Current Implementation (CDN):**
- ✅ Zero bandwidth costs
- ✅ Global edge caching
- ✅ Automatic updates
- ❌ Requires internet access
- ❌ External dependency
- ❌ Potential privacy concerns

**Self-Hosted (Embedded):**
- ✅ Works in air-gapped environments
- ✅ No external dependencies
- ✅ Full control over assets
- ✅ Consistent versioning
- ❌ Larger binary size (~2-3 MB)
- ❌ Manual updates required

## Implementation Options

### Option 1: Use go:embed (Recommended)

Download Swagger UI and embed it in your application:

```bash
# Download Swagger UI
wget https://github.com/swagger-api/swagger-ui/archive/refs/tags/v5.10.0.tar.gz
tar -xzf v5.10.0.tar.gz
mkdir -p static/swagger-ui
cp -r swagger-ui-5.10.0/dist/* static/swagger-ui/
```

Create a custom handler:

```go
package main

import (
    "embed"
    "html/template"
    "io/fs"
    "net/http"
)

//go:embed static/swagger-ui/*
var swaggerAssets embed.FS

func setupSwaggerUI(mux *http.ServeMux, specPath string) error {
    // Serve static assets
    swaggerFS, err := fs.Sub(swaggerAssets, "static/swagger-ui")
    if err != nil {
        return err
    }
    
    fileServer := http.FileServer(http.FS(swaggerFS))
    mux.Handle("/swagger-ui/", http.StripPrefix("/swagger-ui/", fileServer))
    
    // Serve custom index.html
    tmpl := template.Must(template.New("swagger").Parse(swaggerIndexHTML))
    mux.HandleFunc("/swagger", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        tmpl.Execute(w, map[string]string{"SpecPath": specPath})
    })
    
    return nil
}

const swaggerIndexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>API Documentation</title>
    <link rel="stylesheet" type="text/css" href="/swagger-ui/swagger-ui.css">
    <link rel="icon" type="image/png" href="/swagger-ui/favicon-32x32.png">
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="/swagger-ui/swagger-ui-bundle.js"></script>
    <script src="/swagger-ui/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            window.ui = SwaggerUIBundle({
                url: "{{.SpecPath}}",
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout"
            });
        };
    </script>
</body>
</html>`
```

### Option 2: Extend Runtime Handler

Create a custom runtime handler with embedded assets:

```go
package customruntime

import (
    "embed"
    "io/fs"
    "net/http"
    
    "github.com/Infra-Forge/apix/runtime"
)

//go:embed swagger-ui/*
var swaggerAssets embed.FS

type EmbeddedHandler struct {
    *runtime.Handler
    swaggerFS http.FileSystem
}

func NewEmbeddedHandler(cfg runtime.Config) (*EmbeddedHandler, error) {
    baseHandler, err := runtime.NewHandler(cfg)
    if err != nil {
        return nil, err
    }
    
    swaggerFS, err := fs.Sub(swaggerAssets, "swagger-ui")
    if err != nil {
        return nil, err
    }
    
    return &EmbeddedHandler{
        Handler:   baseHandler,
        swaggerFS: http.FS(swaggerFS),
    }, nil
}

func (h *EmbeddedHandler) RegisterHTTP(mux *http.ServeMux) {
    // Register base handler for spec
    h.Handler.RegisterHTTP(mux)
    
    // Serve embedded Swagger UI assets
    mux.Handle("/swagger-ui/", http.StripPrefix("/swagger-ui/", 
        http.FileServer(h.swaggerFS)))
}
```

### Option 3: Use a Third-Party Package

```go
import "github.com/swaggo/http-swagger"

// This package provides embedded Swagger UI
mux.Handle("/swagger/", httpSwagger.Handler(
    httpSwagger.URL("/openapi.json"),
))
```

## Minimal Embedded Solution

For a lightweight solution, create a single-file embedded Swagger UI:

```go
package runtime

import (
    _ "embed"
    "fmt"
)

//go:embed swagger-ui-standalone.html
var swaggerUIStandalone string

func renderSwaggerUIEmbedded(specPath string) string {
    return fmt.Sprintf(swaggerUIStandalone, specPath)
}
```

Create `swagger-ui-standalone.html`:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8" />
    <title>API Documentation</title>
    <style>
        body { margin: 0; padding: 0; }
        #swagger-ui { height: 100vh; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
    <script>
        window.onload = () => {
            window.ui = SwaggerUIBundle({
                url: '%s',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                layout: "StandaloneLayout"
            });
        };
    </script>
</body>
</html>
```

## Configuration Flag

Add a configuration option to choose between CDN and embedded:

```go
type Config struct {
    // ... existing fields ...
    
    // UseEmbeddedSwagger uses embedded Swagger UI assets instead of CDN
    UseEmbeddedSwagger bool
}
```

## Binary Size Comparison

| Method | Binary Size Increase |
|--------|---------------------|
| CDN (current) | +0 KB |
| Embedded (full) | +2-3 MB |
| Embedded (minified) | +1-1.5 MB |
| Single HTML (CDN fallback) | +5-10 KB |

## Recommendation

**For most users:** Stick with the current CDN implementation
- Fast, reliable, and zero overhead
- Swagger UI is updated automatically
- Works for 99% of use cases

**For air-gapped environments:** Use Option 1 (go:embed)
- Full control over assets
- No external dependencies
- Predictable behavior

**For hybrid approach:** Implement both with a config flag
- Default to CDN for convenience
- Allow embedded mode for special cases

## Security Considerations

1. **Subresource Integrity (SRI):** When using CDN, add SRI hashes:
```html
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"
        integrity="sha384-..."
        crossorigin="anonymous"></script>
```

2. **Content Security Policy:** Configure CSP headers:
```go
w.Header().Set("Content-Security-Policy", 
    "default-src 'self'; script-src 'self' https://unpkg.com; style-src 'self' https://unpkg.com")
```

3. **Authentication:** Always protect Swagger UI in production:
```go
if os.Getenv("ENV") == "production" {
    cfg.EnableSwaggerUI = false
}
```

## Future Enhancement

Consider adding this to the library as an optional feature:

```go
import "github.com/Infra-Forge/apix/runtime/embedded"

handler, err := embedded.NewHandler(runtime.Config{
    EnableSwaggerUI: true,
    UseEmbeddedAssets: true, // Use embedded Swagger UI
})
```

This would allow users to opt-in to embedded assets without changing their code significantly.

