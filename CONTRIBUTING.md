# Contributing to Transparency Service

Thank you for your interest in contributing to the Transparency Service! This document provides guidelines and workflows for developers.

## Table of Contents

- [Development Setup](#development-setup)
- [Development Workflow](#development-workflow)
- [Testing Guidelines](#testing-guidelines)
- [Code Style](#code-style)
- [Architecture](#architecture)
- [Pull Request Process](#pull-request-process)
- [Standards Compliance](#standards-compliance)

## Development Setup

### Prerequisites

- [Bun](https://bun.sh) v1.0+ (JavaScript runtime and toolkit)
- Git
- Basic understanding of TypeScript
- Familiarity with IETF standards (COSE, CWT, RFC 6962) is helpful

### Initial Setup

```bash
# Clone the repository
git clone https://github.com/yourusername/transparency-service.git
cd transparency-service

# Install dependencies
bun install

# Run tests to verify setup
bun test

# Initialize local service (optional)
bun run src/cli/index.ts transparency init
```

### Project Structure

```
transparency-service/
├── src/
│   ├── lib/                    # Core library
│   │   ├── cose/              # COSE cryptography (Sign1, keys, hash envelopes)
│   │   ├── merkle/            # Merkle tree (tiles, proofs, checkpoints)
│   │   ├── storage/           # Storage abstraction (local, MinIO, S3, Azure)
│   │   └── database/          # SQLite database layer
│   ├── service/               # HTTP service
│   │   ├── routes/            # API endpoints
│   │   ├── middleware/        # Request/response middleware
│   │   └── types/             # SCRAPI types
│   └── cli/                   # Command-line interface
│       ├── commands/          # CLI commands
│       └── utils/             # CLI utilities
└── tests/
    ├── contract/              # API contract tests (SCRAPI compliance)
    ├── integration/           # End-to-end workflow tests
    ├── interop/               # Cross-implementation compatibility tests (Go/TypeScript)
    ├── performance/           # Performance benchmark tests
    └── unit/                  # Unit tests
```

## Development Workflow

We follow a **Test-Driven Development (TDD)** approach with a strict Red-Green-Refactor cycle:

### TDD Cycle

1. **Red**: Write a failing test first
   ```bash
   # Create test file
   touch tests/unit/my-feature.test.ts

   # Write test
   # Run test - should fail
   bun test tests/unit/my-feature.test.ts
   ```

2. **Green**: Implement minimal code to make test pass
   ```bash
   # Create implementation
   touch src/lib/my-feature.ts

   # Implement feature
   # Run test - should pass
   bun test tests/unit/my-feature.test.ts
   ```

3. **Refactor**: Improve code while keeping tests green
   ```bash
   # Refactor implementation
   # Run all tests - should still pass
   bun test
   ```

### Example TDD Workflow

Let's say you want to add a new feature to validate CWT claims:

```typescript
// Step 1: Write test first (tests/unit/cwt-validation.test.ts)
import { describe, test, expect } from "bun:test";
import { validateCWTClaims } from "../../src/lib/cose/cwt-validation.ts";

describe("CWT Claims Validation", () => {
  test("should validate required issuer claim", () => {
    const claims = { iss: "https://example.com", sub: "test" };
    expect(validateCWTClaims(claims)).toBe(true);
  });

  test("should reject missing issuer claim", () => {
    const claims = { sub: "test" };
    expect(() => validateCWTClaims(claims)).toThrow("Issuer required");
  });
});
```

```typescript
// Step 2: Implement feature (src/lib/cose/cwt-validation.ts)
export function validateCWTClaims(claims: any): boolean {
  if (!claims.iss) {
    throw new Error("Issuer required");
  }
  return true;
}
```

```bash
# Step 3: Run tests
bun test tests/unit/cwt-validation.test.ts
```

## Testing Guidelines

### Test Organization

We use five levels of testing:

1. **Unit Tests** (`tests/unit/`): Test individual functions and modules
2. **Integration Tests** (`tests/integration/`): Test complete workflows
3. **Contract Tests** (`tests/contract/`): Test API compliance with SCRAPI specification
4. **Interop Tests** (`tests/interop/`): Cross-implementation compatibility with Go tlog and COSE
5. **Performance Tests** (`tests/performance/`): Benchmark tests for large files and concurrent operations

**Note**: Test artifacts (`.test-*` directories and `.test-*.db` files) are automatically created during test runs and excluded via `.gitignore`. These should never be committed to the repository.

### Running Tests

```bash
# Run all tests
bun test

# Run specific test file
bun test tests/unit/cose/sign.test.ts

# Run tests matching pattern
bun test --grep "hash envelope"

# Run tests with coverage
bun test --coverage

# Watch mode (re-run on changes)
bun test --watch
```

### Writing Good Tests

**DO:**
- Write descriptive test names that explain what's being tested
- Use arrange-act-assert pattern
- Test edge cases and error conditions
- Keep tests isolated and independent
- Use meaningful variable names

**DON'T:**
- Write tests that depend on external services
- Share state between tests
- Test implementation details (test behavior, not internals)
- Write flaky tests that pass/fail randomly

**Example:**

```typescript
describe("Hash Envelope Creation", () => {
  test("should create hash envelope with SHA-256 digest", async () => {
    // Arrange
    const payload = new TextEncoder().encode("test data");
    const options = { contentType: "text/plain" };
    const key = await generateES256KeyPair();
    const claims = createCWTClaims({ iss: "https://test.com", sub: "test" });

    // Act
    const envelope = await signHashEnvelope(payload, options, key.privateKey, claims);

    // Assert
    expect(envelope.protectedHeaders.get(HASH_ENVELOPE_V1)).toBeDefined();
    const hashEnvelope = envelope.protectedHeaders.get(HASH_ENVELOPE_V1);
    expect(hashEnvelope.payloadHashAlgorithm).toBe(-16); // SHA-256
    expect(hashEnvelope.payloadHash).toBeInstanceOf(Uint8Array);
  });
});
```

### Test Coverage Goals

- **Unit Tests**: 90%+ coverage of core library functions
- **Integration Tests**: Cover all user stories and workflows
- **Contract Tests**: 100% coverage of API endpoints per SCRAPI spec
- **Interop Tests**: 100% compatibility with canonical Go implementations
- **Performance Tests**: Validate all success criteria (10MB < 5s, 1GB < 30s, 100 concurrent)

Current status: **333 tests passing, 6,683 assertions**

## Code Style

### TypeScript Guidelines

- Use TypeScript strict mode
- Always specify return types for functions
- Use interfaces for data structures
- Prefer `const` over `let`, avoid `var`
- Use async/await over Promise chains
- Document complex functions with JSDoc comments

**Example:**

```typescript
/**
 * Create CWT claims for COSE Sign1 protected header
 *
 * @param options - Claim values (issuer, subject, audience)
 * @returns Map containing CWT claims (label 15)
 */
export function createCWTClaims(options: {
  iss: string;
  sub?: string;
  aud?: string;
}): Map<number, any> {
  const claims = new Map<number, any>();

  // Issuer (label 1)
  claims.set(1, options.iss);

  // Subject (label 2)
  if (options.sub) {
    claims.set(2, options.sub);
  }

  // Audience (label 3)
  if (options.aud) {
    claims.set(3, options.aud);
  }

  return claims;
}
```

### Naming Conventions

- **Files**: kebab-case (`hash-envelope.ts`, `merkle-tree.ts`)
- **Functions**: camelCase (`signHashEnvelope`, `getInclusionProof`)
- **Types/Interfaces**: PascalCase (`Receipt`, `ServerContext`)
- **Constants**: UPPER_SNAKE_CASE (`HASH_ENVELOPE_V1`, `SHA256_ALG`)

### Error Handling

Always provide clear error messages with context:

```typescript
// Good
if (!entry) {
  throw new Error(`Entry ${entryId} not found in database`);
}

// Bad
if (!entry) {
  throw new Error("Not found");
}
```

## Architecture

### Core Principles

1. **Standards-First**: Strictly adhere to IETF specifications
2. **Testability**: Design for easy testing (dependency injection, pure functions)
3. **Modularity**: Keep components loosely coupled
4. **Performance**: Optimize for large files and concurrent operations
5. **Security**: Cryptographic operations via Web Crypto API

### Key Design Decisions

#### COSE Layer (`src/lib/cose/`)

Uses CBOR encoding with labeled headers per RFC 8152:

```typescript
// Protected headers (Map)
{
  1: -7,              // alg: ES256
  15: cwtClaims,      // CWT claims (RFC 9597)
  395: hashEnvelope   // Hash envelope
}
```

#### Merkle Tree (`src/lib/merkle/`)

Uses C2SP tile-based format for efficient storage and retrieval:

```
tile/{level}/{index}         // Full tile (256 hashes)
tile/{level}/{index}.p/{w}   // Partial tile (w hashes)
```

#### Database Schema (`src/lib/database/`)

Two main tables:
- `statement_blobs`: Raw COSE Sign1 data (entry_id → bytes)
- `statements`: Searchable metadata (issuer, subject, content type)

### Adding New Features

When adding features, follow this pattern:

1. **Library Layer**: Implement core logic in `src/lib/`
2. **Service Layer**: Add HTTP endpoints in `src/service/routes/`
3. **CLI Layer**: Add commands in `src/cli/commands/`
4. **Tests**: Add unit, integration, and contract tests

## Pull Request Process

### Before Submitting

1. **Run all tests**: `bun test`
2. **Check test coverage**: Ensure new code has tests
3. **Format code**: Follow TypeScript style guidelines
4. **Update documentation**: Add/update JSDoc comments and README if needed

### PR Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Added unit tests
- [ ] Added integration tests
- [ ] All tests passing (327+ tests)

## Standards Compliance
- [ ] Follows IETF specifications (SCITT, COSE, CWT, RFC 6962)
- [ ] No breaking changes to API contracts

## Checklist
- [ ] Code follows TypeScript style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No console.log or debug code
```

### Review Process

1. Submit PR with descriptive title and description
2. Automated tests run (must pass)
3. Code review by maintainers
4. Address feedback and update PR
5. Approval and merge

## Standards Compliance

This project implements multiple IETF standards. When contributing, ensure compliance:

### COSE (RFC 8152)

- Use labeled headers per specification
- ES256 algorithm (ECDSA with P-256 and SHA-256)
- COSE Sign1 structure: `[protected, unprotected, payload, signature]`

### CWT (RFC 8392)

- Standard claims: iss (1), sub (2), aud (3)
- Claims in COSE header per RFC 9597 (label 15)

### RFC 6962 (Merkle Trees)

- Left-balanced binary trees
- SHA-256 hashing with domain separation (`\x00` for leaves, `\x01` for nodes)
- Inclusion and consistency proofs

### SCITT

- Transparent statements with receipts
- Append-only log guarantees
- SCRAPI HTTP endpoint compliance

## Getting Help

- **Issues**: Open a GitHub issue for bugs or feature requests
- **Discussions**: Use GitHub Discussions for questions
- **Documentation**: See `docs/` directory and inline code comments
- **Examples**: Check `tests/` for usage examples

## License

By contributing, you agree that your contributions will be licensed under the same license as the project (MIT License).

---

Thank you for contributing to the Transparency Service!
