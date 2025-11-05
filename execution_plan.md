# Execution Plan — apix Go API Documentation Library

## Context & Research Findings

### Why a New Library
- Current Go OpenAPI tooling (swaggo, spec-first generators) relies on comments or YAML edits, allowing AI agents and developers to drift from code.
- We need code-first, deterministic OpenAPI 3.1 generation with runtime + go:generate workflows, aligned with 2025 security/DX standards.

### Reflection Limits & Overrides
- **Spring Boot**: Uses runtime annotations and allows explicit overrides via `@Schema`, `OpenApiCustomiser` when reflection falls short.
- **FastAPI**: Relies on type hints (Pydantic); complex types require explicit Pydantic models or `Annotated` metadata.
- **Go ecosystem**: swaggo uses comment overrides; goa/ogen are DSL/spec-first. Kin-openapi is the building block.
- **adopted strategy**:
  - Route options `WithRequestOverride`, `WithResponseOverride`, etc. to define schema/content/headers/examples manually when necessary.
  - Project-wide customizers (e.g., `RegisterCustomizer`) to mirror Spring's OpenApiCustomiser.
  - Config file (future milestone) to map `method:path` to overrides without code changes.

## Implementation Status

### ✅ Milestone 1 (v0.1) - COMPLETE
1. **Core Registry** ✅ - Thread-safe route metadata storage, deterministic snapshot, success status, security, headers
2. **Echo Adapter** ✅ - Typed registration helpers, custom decoders, validation hooks (82.5% coverage)
3. **Spec Builder** ✅ - OpenAPI 3.1 generation with kin-openapi, deterministic output, 201/401/403 defaults (77.9% coverage)
4. **CLI** ✅ - `apix generate` and `spec-guard` commands with DO-NOT-EDIT header (73% coverage)
5. **Runtime Endpoints** ✅ - `/openapi.json` + optional Swagger UI, caching (81% coverage)
6. **Tests** ✅ - Comprehensive test suite, 84% overall coverage

### ✅ Milestone 2 (v0.2) - COMPLETE
**Completed:**
1. **Chi Adapter** ✅ - Full Chi router integration with typed handlers (88% coverage)
2. **Gorilla/Mux Adapter** ✅ - Full Mux router integration with typed handlers (88% coverage)
3. **Gin Adapter** ✅ - Full Gin framework integration with typed handlers (87.5% coverage)
4. **Fiber Adapter** ✅ - Full Fiber v3 framework integration with typed handlers (86.9% coverage)
5. **Shared Error Schema** ✅ - Standard ErrorResponse with 4xx/5xx helpers (100% coverage)
6. **Golden Tests** ✅ - Comprehensive CRUD spec validation and determinism tests
7. **Integration Tests** ✅ - Full integration tests for all 5 framework adapters

### 🔮 Milestone 3 (v0.3) - PLANNED
**Core Features:**
- **Typed Query/Header Parameters** - Struct-based parameter extraction with type inference
- **Middleware Auto-detection** - Automatic security scheme detection from middleware stack
- **DX Extras** - Pagination headers (Link, X-Total-Count), ETag support
- **Example Applications** - Complete working examples for all 5 frameworks

**Additional Features:**
- Structured examples via tags/helpers
- Additional content types (multipart/form-data, form-urlencoded)
- Plugin hooks for custom metadata
- Observability (logging, metrics)
- Migration guide for swaggo users

## Immediate Next Steps (Milestone 2 Completion)

### Priority 1: Core Features
1. **Shared Error Schema**
   - Define standard `ErrorResponse` struct
   - Auto-inject for 4xx/5xx responses
   - Allow customization via builder options

2. **Typed Query/Header Parameters**
   - Support struct-based query parameter extraction
   - Infer parameter types from struct tags
   - Generate OpenAPI parameter definitions

3. **Example Application**
   - Create `examples/echo/`, `examples/chi/`, `examples/mux/`
   - Demonstrate CRUD operations, security, validation
   - Include `go:generate` workflow and CI integration

### Priority 2: DX Enhancements
4. **Pagination Headers**
   - Auto-detect pagination patterns (offset/limit, cursor)
   - Document Link, X-Total-Count headers
   - Helper functions for common pagination schemes

5. **Middleware Auto-detection**
   - Scan middleware stack for JWT/Bearer auth
   - Auto-inject security schemes
   - Support custom middleware detection hooks

## Risks & Mitigations
- Reflection misses → mitigated by override options/customizers ✅
- Agent spec edits → enforced by guardrails (CI, MCP `spec_guard`, DO-NOT-EDIT header) ✅
- Performance on large services → caching + go:generate workflow ✅
- Adoption friction → wrappers support incremental migration; documentation and examples prioritized ⏳


