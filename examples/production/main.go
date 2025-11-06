package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Infra-Forge/apix"
	"github.com/Infra-Forge/apix/openapi"
	"github.com/Infra-Forge/apix/runtime"
	chiadapter "github.com/Infra-Forge/apix/chi"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/time/rate"
)

// Config holds application configuration
type Config struct {
	Env        string
	Port       string
	APIVersion string
	ServerURL  string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() Config {
	return Config{
		Env:        getEnv("ENV", "development"),
		Port:       getEnv("PORT", "8080"),
		APIVersion: getEnv("API_VERSION", "1.0.0"),
		ServerURL:  getEnv("SERVER_URL", "http://localhost:8080"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Response types
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Env     string `json:"env"`
}

type ItemResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CreateItemRequest struct {
	Name string `json:"name" validate:"required"`
}

// Handlers
func healthHandler(ctx context.Context, req apix.NoBody) (HealthResponse, error) {
	cfg := ctx.Value("config").(Config)
	return HealthResponse{
		Status:  "healthy",
		Version: cfg.APIVersion,
		Env:     cfg.Env,
	}, nil
}

func createItemHandler(ctx context.Context, req CreateItemRequest) (ItemResponse, error) {
	return ItemResponse{
		ID:   "item-123",
		Name: req.Name,
	}, nil
}

func getItemHandler(ctx context.Context, req apix.NoBody) (ItemResponse, error) {
	return ItemResponse{
		ID:   "item-123",
		Name: "Sample Item",
	}, nil
}

// Rate limiter middleware
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

// Context middleware to inject config
func configMiddleware(cfg Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), "config", cfg)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func main() {
	// Load configuration
	cfg := LoadConfig()
	
	log.Printf("Starting application in %s mode", cfg.Env)
	log.Printf("Version: %s", cfg.APIVersion)

	// Reset registry for clean state
	apix.ResetRegistry()

	// Create Chi router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(configMiddleware(cfg))

	// Timeout middleware
	r.Use(middleware.Timeout(60 * time.Second))

	// Create apix adapter
	adapter := chiadapter.New(r)

	// Register routes
	chiadapter.Get(adapter, "/health", healthHandler,
		apix.WithSummary("Health check"),
		apix.WithDescription("Returns the health status of the API"),
		apix.WithTags("System"),
	)

	chiadapter.Post(adapter, "/api/items", createItemHandler,
		apix.WithSummary("Create item"),
		apix.WithDescription("Create a new item"),
		apix.WithTags("Items"),
		apix.WithSecurity("BearerAuth", "items:write"),
		apix.WithStandardErrors(),
	)

	chiadapter.Get(adapter, "/api/items/{id}", getItemHandler,
		apix.WithSummary("Get item"),
		apix.WithDescription("Get an item by ID"),
		apix.WithTags("Items"),
		apix.WithParameter(apix.Parameter{
			Name:        "id",
			In:          "path",
			Required:    true,
			SchemaType:  "string",
			Description: "Item ID",
		}),
		apix.WithNotFoundError("Item not found"),
	)

	// Create runtime handler with environment-specific config
	runtimeCfg := runtime.Config{
		Title:           "Production API Example",
		Version:         cfg.APIVersion,
		Format:          "json",
		Validate:        true,
		EnableSwaggerUI: cfg.Env != "production", // Disable Swagger UI in production
		Servers:         []string{cfg.ServerURL},
		CustomizeBuilder: func(b *openapi.Builder) {
			b.Info.Description = "Production-ready API with apix"
			b.Info.Contact = &openapi3.Contact{
				Name:  "API Support",
				Email: "support@example.com",
				URL:   "https://example.com/support",
			}
			b.Info.License = &openapi3.License{
				Name: "MIT",
				URL:  "https://opensource.org/licenses/MIT",
			}

			// Security schemes
			b.SecuritySchemes = openapi3.SecuritySchemes{
				"BearerAuth": &openapi3.SecuritySchemeRef{
					Value: openapi3.NewJWTSecurityScheme(),
				},
			}
		},
	}

	// Set cache TTL based on environment
	switch cfg.Env {
	case "production":
		runtimeCfg.CacheTTL = 10 * time.Minute
	case "staging":
		runtimeCfg.CacheTTL = 1 * time.Minute
	default:
		runtimeCfg.CacheTTL = 0 // No cache in development
	}

	handler, err := runtime.NewHandler(runtimeCfg)
	if err != nil {
		log.Fatalf("Failed to create runtime handler: %v", err)
	}

	// Register OpenAPI endpoints
	mux := http.NewServeMux()
	handler.RegisterHTTP(mux)
	
	// Apply rate limiting to OpenAPI endpoints
	r.Handle("/openapi.json", rateLimitMiddleware(mux))
	if cfg.Env != "production" {
		r.Handle("/swagger", rateLimitMiddleware(mux))
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", cfg.Port)
		log.Printf("OpenAPI spec: http://localhost:%s/openapi.json", cfg.Port)
		if cfg.Env != "production" {
			log.Printf("Swagger UI: http://localhost:%s/swagger", cfg.Port)
		}
		log.Printf("Health check: http://localhost:%s/health", cfg.Port)
		
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

