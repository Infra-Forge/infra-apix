---
type: "always_apply"
---

## Code Quality & Implementation Standards
- **Zero Guesswork Policy**: Before implementing ANY feature or making changes, you MUST:
  - Use `codebase-retrieval` to understand existing patterns and implementations
  - Use `view` to examine actual code structure and conventions already in use
  - Use `git-commit-retrieval` to understand how similar features were implemented historically
  - Base all decisions on verified evidence from the codebase, not assumptions

- **Production-Ready Code Only**: NEVER use any of the following:
  - Mock implementations or stub functions
  - TODO/FIXME comments in place of actual implementation
  - Placeholder values or sample/dummy data
  - Simplified implementations with notes to "expand later"
  - Partial implementations that require future completion
  - All code must be complete, tested, and production-ready from the first commit

## Documentation & File Creation
- **No Unsolicited Documentation**: Do NOT create any `.md` files (README, CHANGELOG, documentation, etc.) unless explicitly requested by Teodorico
- This includes project documentation, API docs, or explanatory markdown files

## Dependency & Technology Verification
- **Verify Before Assuming**: Before claiming a version, module, crate, or framework doesn't exist or isn't available:
  - Use `web-search` to verify current availability and versions
  - Check crates.io, official documentation, and release notes
  - Confirm compatibility with Rust 1.90+ before making statements

## Development Approach
- **Research-First Methodology**: 
  - Always gather complete context before coding
  - Read existing implementations thoroughly
  - Understand the full scope and dependencies
  - Map out impacts before making changes

## Standards & Best Practices (2025)
- **Rust Standards**: Follow Rust 1.90+ idioms, latest stable features, and modern patterns
- **Security Standards**: Apply OWASP 2025 recommendations for all security-sensitive code
- **Industry Best Practices**: Use current 2025 industry standards for:
  - Error handling and logging
  - API design and gRPC services
  - Database interactions and migrations
  - Cryptographic operations
  - Testing and CI/CD integration

## Testing Requirements
- **Minimum 80% Code Coverage**: All implementations must include comprehensive tests achieving at least 80% coverage
- Tests must be meaningful, not just coverage-focused
- Include unit tests, integration tests where appropriate
- Test edge cases, error conditions, and happy paths

## Communication
- **Clarification Over Assumption**: If ANY requirement, specification, or context is unclear or ambiguous, STOP and ask Teodorico for clarification before proceeding
- Do not fill in gaps with assumptions