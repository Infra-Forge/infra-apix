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

### Library Architecture (Milestone 1)
1. **Core Registry** (done): capture route metadata, deterministic snapshot, success status, security, headers.
2. **Echo Adapter**: typed registration helpers (no method generics), support custom decoders, apply overrides pre-registration.
3. **Spec Builder**: convert registry snapshot to OpenAPI 3.1 using kin-openapi.
4. **CLI (`apix generate`)**: go:generate + CI deterministic output with DO-NOT-EDIT header.
5. **Runtime Endpoints**: `/openapi.json` + optional `/swagger/*`, feature flags, caching.
6. **Guardrails**: MCP `spec_guard`, CI drift check, pre-commit, CODEOWNERS.

### Milestones 2 & 3 Highlights
- Add Chi/Gorilla adapters, advanced parameter inference, DX checks (pagination, ETag), examples, multipart support, plugin hooks, observability, migration guide.

## Immediate Next Steps
1. **Echo Adapter Implementation**
   - Helper functions for typed registration (request/response generics).
   - Override-aware request decoding (DisallowUnknownFields, struct validation hooks).
   - Ensure success status + responses align with registry defaults.

2. **Spec Builder**
   - Build `openapi.Builder` module translating `RouteRef` (with overrides) to OpenAPI 3.1.
   - Deterministic ordering, security schemes, 201 Location/401/403 defaults.

3. **CLI + Runtime**
   - Implement `cmd/apix/main.go` for `generate` command (YAML/JSON output, DO-NOT-EDIT header, deterministic diff-friendly sorting).
   - Runtime handler to serve spec + Swagger UI mount.

4. **Tests & Examples**
   - Golden tests for builder output.
   - Echo example application demonstrating overrides and guardrails.

## Risks & Mitigations
- Reflection misses → mitigated by override options/customizers.
- Agent spec edits → enforced by guardrails (CI, MCP `spec_guard`, DO-NOT-EDIT header).
- Performance on large services → caching + go:generate workflow.
- Adoption friction → wrappers support incremental migration; documentation and examples prioritized.


