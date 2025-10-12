# Migration Mapping: Existing Code to Monorepo Structure

**Generated**: 2025-10-12
**Purpose**: Map existing TypeScript implementation to new monorepo structure and identify what needs to be created vs. migrated

## Executive Summary

**Existing TypeScript Implementation Status**: ✅ Substantial (33 source files + 25+ test files)
**Existing Go Implementation Status**: ⚠️ Limited (test vector generators only)

### Key Findings:

1. **TypeScript has extensive implementation** covering all core modules:
   - COSE signing/verification
   - Merkle tree (tile-log) operations
   - SQLite database layer
   - HTTP server with full SCRAPI routes
   - CLI with `transparency init` and `transparency serve`
   - Comprehensive test suite (unit, contract, integration, interop)

2. **Go implementation is minimal**:
   - Test vector generators in `tests/interop/go-tlog-generator/` (8 files)
   - COSE vector generator in `tests/interop/go-cose-generator/`
   - **Per Constitution Principle VIII**: Go is the "canonical reference" for interoperability
   - **Recommendation**: Use `github.com/veraison/go-cose` for COSE implementation

3. **Migration Strategy**:
   - **Phase 2 TypeScript tasks (T009, T011)**: Already implemented, just need migration
   - **Phase 2 Go tasks (T010, T012)**: Need to be created from scratch
   - **Phase 3 (T013)**: Critical - migrate existing TypeScript code to `scitt-typescript/`
   - **Phase 3 (T014-T027)**: Build Go implementation using existing TS as reference

---

## Existing TypeScript Structure

### Source Files (src/)

```
src/
├── cli/
│   ├── index.ts                    # Main CLI entry point
│   ├── commands/
│   │   ├── transparency/
│   │   │   ├── init.ts
│   │   │   └── serve.ts
│   │   └── issuer/
│   │       └── ...
├── lib/
│   ├── cose/                       # ✅ COMPLETE COSE implementation
│   │   ├── constants.ts
│   │   ├── hash-envelope.ts
│   │   ├── issuer-resolver.ts
│   │   ├── key-material.ts         # T009 equivalent (ES256 key generation)
│   │   ├── sign.ts
│   │   └── signer.ts
│   ├── database/                   # ✅ COMPLETE database layer
│   │   ├── log-state.ts
│   │   ├── merkle.ts
│   │   ├── receipts.ts
│   │   ├── schema.ts               # T011 equivalent (SQLite schema)
│   │   └── statements.ts
│   ├── merkle/                     # ✅ COMPLETE Merkle tree (tile-log)
│   │   ├── checkpoints.ts
│   │   ├── proofs.ts
│   │   ├── tile-log.ts
│   │   └── tile-naming.ts
│   └── storage/                    # ✅ Storage abstraction
│       ├── interface.ts
│       └── local.ts
├── service/                        # ✅ COMPLETE HTTP server
│   ├── server.ts                   # Main Bun server
│   └── routes/
│       ├── checkpoint.ts
│       ├── config.ts
│       ├── health.ts
│       ├── receipts.ts
│       ├── register.ts
│       └── tiles.ts
└── types/                          # TypeScript type definitions
    ├── cose.ts
    ├── database.ts
    └── storage.ts
```

### Test Files (tests/)

```
tests/
├── unit/
│   ├── cose/                       # 5 test files
│   │   ├── hash-envelope.test.ts
│   │   ├── issuer-resolver.test.ts
│   │   ├── key-material.test.ts
│   │   ├── sign.test.ts
│   │   └── signer.test.ts
│   ├── database/                   # 4 test files
│   │   ├── log-state.test.ts
│   │   ├── receipts.test.ts
│   │   ├── schema.test.ts
│   │   └── statements.test.ts
│   ├── merkle/                     # 4 test files
│   │   ├── checkpoints.test.ts
│   │   ├── proofs.test.ts
│   │   ├── tile-log.test.ts
│   │   └── tile-naming.test.ts
│   └── storage/                    # 2 test files
│       ├── interface.test.ts
│       └── local.test.ts
├── contract/                       # 5+ test files (API contracts)
│   ├── config.test.ts
│   ├── health.test.ts
│   ├── receipts.test.ts
│   ├── registration.test.ts
│   ├── service-keys.test.ts
│   └── tiles.test.ts
├── integration/                    # End-to-end tests
│   └── auditor-workflow.test.ts
├── interop/                        # Cross-implementation tests
│   ├── go-cose-generator/
│   │   └── generate_vectors.go
│   ├── go-tlog-generator/          # 8 Go files (existing)
│   │   ├── debug_indexes.go
│   │   ├── debug_proof.go
│   │   ├── gen_correct_vectors.go
│   │   ├── generate_vectors.go
│   │   ├── main.go
│   │   ├── proper_reader.go
│   │   ├── test_check_tree.go
│   │   └── verify_ts_vectors.go
│   ├── go-interop.test.ts
│   ├── go-cose-interop.test.ts
│   └── tile-format.test.ts
└── performance/
    └── registration-performance.test.ts
```

---

## Migration Plan: Phase 2 Tasks (T009-T012)

### T009 [P] - TypeScript ES256 Key Generation
**Status**: ✅ Already implemented
**Location**: `src/lib/cose/key-material.ts:25-34`
**Migration**:
- Copy `src/lib/cose/` → `scitt-typescript/src/lib/cose/`
- Copy `tests/unit/cose/` → `scitt-typescript/tests/unit/cose/`

**Key Functions**:
```typescript
export async function generateES256KeyPair(): Promise<CryptoKeyPair>
export async function exportPublicKeyToJWK(publicKey: CryptoKey): Promise<JWK>
export async function exportPrivateKeyToPEM(privateKey: CryptoKey): Promise<string>
export async function importPrivateKeyFromPEM(pem: string): Promise<CryptoKey>
```

### T010 [P] - Go ES256 Key Generation
**Status**: ❌ Needs to be created
**Implementation Plan**:
- Use standard library `crypto/ecdsa` and `crypto/elliptic`
- Curve: `elliptic.P256()`
- PEM encoding: `x509.MarshalPKCS8PrivateKey` / `x509.MarshalPKIXPublicKey`
- Reference: `github.com/veraison/go-cose` for COSE_Key conversions

**Target Location**: `scitt-golang/pkg/cose/keygen.go`

### T011 [P] - SQLite Schema
**Status**: ✅ Already implemented (dual format)
**Locations**:
1. `docs/schema.sql` - Created in Phase 1 (T004)
2. `src/lib/database/schema.ts:12-159` - Programmatic TypeScript version

**Migration**:
- Keep `docs/schema.sql` as shared reference
- Copy `src/lib/database/` → `scitt-typescript/src/lib/database/`
- Create Go version in `scitt-golang/pkg/database/schema.go`

**Schema Highlights**:
- `statements`: Statement metadata with tree position
- `receipts`: Receipt pointers to storage
- `tiles`: Merkle tree tile metadata
- `tree_state`: Current Merkle tree state
- `current_tree_size`: Singleton for tree size
- `service_config`: Configuration key-value store
- `service_keys`: Transparency service signing keys

### T012 [P] - MinIO Bucket Structure
**Status**: ❌ Needs documentation
**Current Implementation**: Local storage only (`src/lib/storage/local.ts`)
**Action Required**:
- Document MinIO bucket structure in `docs/storage/minio.md`
- Define bucket naming conventions
- Document object key formats for receipts, tiles, checkpoints
- Create setup guide for MinIO

---

## Migration Plan: Phase 3 Tasks (T013-T027)

### T013 [P] - Migrate TypeScript to Monorepo
**Status**: ⚠️ CRITICAL - Must happen before Phase 3 Go implementation
**Git History Preservation**: Use `git mv` to preserve history

**Migration Steps**:
```bash
# 1. Move source files
git mv src/cli scitt-typescript/src/
git mv src/lib scitt-typescript/src/
git mv src/service scitt-typescript/src/
git mv src/types scitt-typescript/src/

# 2. Move test files
git mv tests/unit scitt-typescript/tests/
git mv tests/contract scitt-typescript/tests/
git mv tests/integration scitt-typescript/tests/
git mv tests/performance scitt-typescript/tests/

# 3. Keep interop tests at root (shared)
# tests/interop/ stays in place

# 4. Update package.json
git mv package.json scitt-typescript/
# Then merge with scitt-typescript/package.json from Phase 1

# 5. Update tsconfig.json
git mv tsconfig.json scitt-typescript/

# 6. Update imports in all files
# Change relative imports to reflect new structure
```

**Dependencies to Migrate**:
- `cbor-x`: ^1.6.0 (CBOR encoding/decoding)
- `@aws-sdk/client-s3`: ^3.621.0 (S3/MinIO client)

### T014-T027 - Go Implementation Tasks
**Status**: ❌ All need to be created from scratch
**Strategy**: Use migrated TypeScript as reference implementation

**Parallel Structure**:
```
scitt-golang/
├── pkg/
│   ├── cose/              # T010, T015, T016
│   ├── database/          # T011, T017, T018
│   ├── merkle/            # T019, T020, T021
│   └── storage/           # T012, T022
├── cmd/
│   ├── scitt/             # T023 (CLI)
│   └── scitt-server/      # T024 (HTTP server)
└── tests/
    ├── unit/              # T025
    ├── contract/          # T026
    └── integration/       # T027
```

**Key Go Dependencies**:
- `github.com/veraison/go-cose` - COSE implementation (user provided)
- `github.com/fxamacker/cbor/v2` - CBOR encoding
- `modernc.org/sqlite` - Pure Go SQLite
- `github.com/minio/minio-go/v7` - MinIO client
- `golang.org/x/mod/sumdb/tlog` - Tile-log (C2SP canonical implementation)

---

## Task Status Matrix

| Phase | Task | TypeScript Status | Go Status | Action Required |
|-------|------|-------------------|-----------|-----------------|
| 2 | T009 | ✅ Implemented | N/A | Migrate to `scitt-typescript/` |
| 2 | T010 | N/A | ❌ Create | Implement using `crypto/ecdsa` |
| 2 | T011 | ✅ Implemented | ❌ Create | Migrate TS, create Go version |
| 2 | T012 | ⚠️ Partial | ❌ Document | Document MinIO structure |
| 3 | T013 | ⚠️ **CRITICAL** | N/A | **Execute git mv migration** |
| 3 | T014 | ✅ Reference exists | ❌ Create | Port TS cose/sign.ts to Go |
| 3 | T015 | ✅ Reference exists | ❌ Create | Port TS cose/signer.ts to Go |
| 3 | T016 | ✅ Reference exists | ❌ Create | Port TS cose/hash-envelope.ts to Go |
| 3 | T017 | ✅ Reference exists | ❌ Create | Port TS database layer to Go |
| 3 | T018 | ✅ Reference exists | ❌ Create | Port TS database/statements.ts to Go |
| 3 | T019 | ✅ Reference exists | ❌ Create | Use `golang.org/x/mod/sumdb/tlog` |
| 3 | T020 | ✅ Reference exists | ❌ Create | Port TS merkle/proofs.ts to Go |
| 3 | T021 | ✅ Reference exists | ❌ Create | Port TS merkle/checkpoints.ts to Go |
| 3 | T022 | ✅ Reference exists | ❌ Create | Port TS storage layer to Go |
| 3 | T023 | ✅ Reference exists | ❌ Create | Port TS CLI to Go (cobra?) |
| 3 | T024 | ✅ Reference exists | ❌ Create | Port TS server to Go (net/http) |
| 3 | T025 | ✅ Reference exists | ❌ Create | Port unit tests to Go |
| 3 | T026 | ✅ Reference exists | ❌ Create | Port contract tests to Go |
| 3 | T027 | ✅ Reference exists | ❌ Create | Port integration tests to Go |

---

## Critical Path Forward

### Immediate Next Steps (In Order):

1. **Execute T013**: Migrate existing TypeScript code to `scitt-typescript/`
   - Preserves git history
   - Establishes TypeScript baseline in monorepo
   - **Blocks**: All Phase 3 Go tasks (T014-T027)

2. **Complete T012**: Document MinIO bucket structure
   - Required for storage layer implementation
   - **Blocks**: T022 (Go storage), T024 (Go server)

3. **Execute T010**: Create Go ES256 key generation
   - Foundation for Go COSE implementation
   - **Blocks**: T014-T016 (Go COSE modules)

4. **Execute T014-T027**: Build Go implementation
   - Use migrated TypeScript as reference
   - Maintain API equivalence per FR-007
   - Generate cross-implementation test vectors

### Risk Assessment:

- **HIGH RISK**: Skipping T013 migration would duplicate effort and lose git history
- **MEDIUM RISK**: Go implementation without TypeScript reference would miss edge cases
- **LOW RISK**: Test suite migration - existing tests provide excellent coverage

---

## Verification Criteria

### Post-Migration Checklist:

- [ ] All TypeScript source files moved to `scitt-typescript/src/`
- [ ] All TypeScript tests moved to `scitt-typescript/tests/`
- [ ] Git history preserved for all moved files
- [ ] TypeScript tests pass in new location: `cd scitt-typescript && bun test`
- [ ] Interop tests remain in `tests/interop/` (shared)
- [ ] CI workflows trigger correctly for TypeScript changes
- [ ] Package.json scripts work in new location

### Interoperability Requirements (per Constitution):

- [ ] Go implementation uses `golang.org/x/mod/sumdb/tlog` (Principle VIII)
- [ ] Go implementation uses `github.com/veraison/go-cose` for COSE
- [ ] Both implementations produce identical receipts (FR-007)
- [ ] Both implementations produce identical checkpoints (FR-007)
- [ ] Both implementations can verify each other's signatures (FR-006)
- [ ] Cross-implementation test vectors pass in both directions

---

## Appendix: Key File Mappings

### TypeScript → scitt-typescript/

| Current Location | New Location |
|------------------|--------------|
| `src/cli/index.ts` | `scitt-typescript/src/cli/index.ts` |
| `src/lib/cose/key-material.ts` | `scitt-typescript/src/lib/cose/key-material.ts` |
| `src/lib/database/schema.ts` | `scitt-typescript/src/lib/database/schema.ts` |
| `src/lib/merkle/tile-log.ts` | `scitt-typescript/src/lib/merkle/tile-log.ts` |
| `src/service/server.ts` | `scitt-typescript/src/service/server.ts` |
| `tests/unit/cose/key-material.test.ts` | `scitt-typescript/tests/unit/cose/key-material.test.ts` |
| `package.json` | `scitt-typescript/package.json` (merge) |
| `tsconfig.json` | `scitt-typescript/tsconfig.json` |

### Go → scitt-golang/ (to be created)

| Module | Target Location |
|--------|-----------------|
| ES256 key generation | `scitt-golang/pkg/cose/keygen.go` |
| COSE Sign1 | `scitt-golang/pkg/cose/sign.go` |
| Database schema | `scitt-golang/pkg/database/schema.go` |
| Merkle tree | `scitt-golang/pkg/merkle/tlog.go` |
| CLI | `scitt-golang/cmd/scitt/main.go` |
| HTTP server | `scitt-golang/cmd/scitt-server/main.go` |

---

**Next Action**: Execute T013 - Migrate TypeScript code to monorepo structure
