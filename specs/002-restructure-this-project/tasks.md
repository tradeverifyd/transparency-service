# Implementation Tasks: Dual-Language Project Restructure

**Feature Branch**: `002-restructure-this-project`
**Date**: 2025-10-12
**Total Tasks**: 86
**Estimated MVP**: User Story 1 (Tasks T001-T027)

## Task Organization

Tasks are organized by user story priority to enable incremental delivery:
- **Phase 1**: Setup & Infrastructure (T001-T008)
- **Phase 2**: Foundational Prerequisites (T009-T012)
- **Phase 3**: User Story 1 - Language-Independent Quick Start [P1] (T013-T027)
- **Phase 4**: User Story 2 - Cross-Implementation Interoperability [P1] (T028-T044)
- **Phase 5**: User Story 3 - Consistent CLI Operations [P2] (T045-T056)
- **Phase 6**: User Story 4 - Server Deployment Parity [P2] (T057-T072)
- **Phase 7**: User Story 5 - Library Integration [P3] (T073-T078)
- **Phase 8**: Polish & Integration (T079-T086)

**Test Strategy**: Cross-implementation tests validate interoperability per Constitution Principle III (Test-First Development)

---

## Phase 1: Setup & Infrastructure

**Goal**: Initialize monorepo structure and shared tooling

### T001: Create monorepo directory structure
**Story**: Setup
**Description**: Create top-level directories for dual-language implementation
```bash
mkdir -p scitt-typescript scitt-golang tests/interop docs/.github/workflows
```
**Files**:
- `/scitt-typescript/` (new directory)
- `/scitt-golang/` (new directory)
- `/tests/interop/` (new directory)

**Acceptance**: Directory structure matches plan.md specification

---

### T002 [P]: Initialize TypeScript project
**Story**: Setup
**Description**: Create TypeScript project with Bun configuration
```bash
cd scitt-typescript
bun init
```
**Files**:
- `/scitt-typescript/package.json`
- `/scitt-typescript/tsconfig.json`
- `/scitt-typescript/bun.lockb`

**Acceptance**: `bun install` completes successfully

---

### T003 [P]: Initialize Go project
**Story**: Setup
**Description**: Create Go module with proper naming
```bash
cd scitt-golang
go mod init github.com/org/transparency-service/scitt-golang
```
**Files**:
- `/scitt-golang/go.mod`
- `/scitt-golang/go.sum`

**Acceptance**: `go build ./...` completes without errors

---

### T004: Create shared documentation structure
**Story**: Setup
**Description**: Set up docs directory with OpenAPI spec and install scripts
**Files**:
- `/docs/openapi.yaml` (copy from specs/002-.../contracts/openapi.yaml)
- `/docs/schema.sql` (new, SQLite schema)
- `/docs/install/install-typescript.sh` (new)
- `/docs/install/install-golang.sh` (new)

**Acceptance**: OpenAPI spec validates with swagger-parser

---

### T005 [P]: Create GitHub Actions workflow structure
**Story**: Setup
**Description**: Initialize CI/CD workflows for monorepo
**Files**:
- `/.github/workflows/ci-typescript.yml`
- `/.github/workflows/ci-golang.yml`
- `/.github/workflows/ci-interop.yml`
- `/.github/workflows/release.yml`

**Acceptance**: Workflows trigger on correct paths

---

### T006: Update root .gitignore
**Story**: Setup
**Description**: Update gitignore for monorepo with dual languages
**Files**:
- `/.gitignore` (update existing)

**Content**:
```
# TypeScript
scitt-typescript/node_modules/
scitt-typescript/dist/
scitt-typescript/*.tsbuildinfo

# Go
scitt-golang/vendor/
scitt-golang/*.exe
scitt-golang/*.test

# Shared
*.db
*.db-*
tests/.test-*/
```

**Acceptance**: Git status clean after build artifacts

---

### T007 [P]: Create parallel directory structures (TypeScript)
**Story**: Setup
**Description**: Create library/, cli/, server/, tests/ in scitt-typescript/
**Files**:
- `/scitt-typescript/src/lib/` (directory)
- `/scitt-typescript/src/cli/` (directory)
- `/scitt-typescript/src/server/` (directory)
- `/scitt-typescript/tests/unit/` (directory)
- `/scitt-typescript/tests/integration/` (directory)
- `/scitt-typescript/tests/contract/` (directory)

**Acceptance**: Directory structure matches FR-012

---

### T008 [P]: Create parallel directory structures (Go)
**Story**: Setup
**Description**: Create pkg/, cmd/, internal/, tests/ in scitt-golang/
**Files**:
- `/scitt-golang/pkg/` (directory)
- `/scitt-golang/cmd/scitt/` (directory)
- `/scitt-golang/cmd/scitt-server/` (directory)
- `/scitt-golang/internal/` (directory)
- `/scitt-golang/tests/unit/` (directory)
- `/scitt-golang/tests/integration/` (directory)
- `/scitt-golang/tests/contract/` (directory)

**Acceptance**: Directory structure matches FR-012

---

## Phase 2: Foundational Prerequisites

**Goal**: Core libraries and schemas that block all user stories

### T009 [P]: Implement ES256 key generation (TypeScript)
**Story**: Foundation
**Description**: Core cryptographic key generation using Web Crypto API
**Files**:
- `/scitt-typescript/src/lib/keys/generate.ts`
- `/scitt-typescript/src/lib/keys/cose-key.ts`
- `/scitt-typescript/src/lib/keys/index.ts`

**Implementation**:
```typescript
export async function generateES256Key(): Promise<CoseKeyPair> {
  const keyPair = await crypto.subtle.generateKey(
    { name: "ECDSA", namedCurve: "P-256" },
    true,
    ["sign", "verify"]
  );
  return toCoseKey(keyPair);
}
```

**Acceptance**: Generates valid ES256 COSE Key format

---

### T010 [P]: Implement ES256 key generation (Go)
**Story**: Foundation
**Description**: Core cryptographic key generation using crypto/ecdsa
**Files**:
- `/scitt-golang/pkg/keys/generate.go`
- `/scitt-golang/pkg/keys/cose_key.go`
- `/scitt-golang/pkg/keys/keys.go`

**Implementation**:
```go
func GenerateES256Key() (*CoseKeyPair, error) {
    privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    if err != nil {
        return nil, err
    }
    return ToCoseKey(privateKey)
}
```

**Acceptance**: Generates valid ES256 COSE Key format (interoperable with T009)

---

### T011: Create SQLite schema
**Story**: Foundation
**Description**: Database schema for statements, checkpoints, issuers
**Files**:
- `/docs/schema.sql`

**Content** (from data-model.md):
```sql
CREATE TABLE statements (
  entry_id TEXT PRIMARY KEY,
  cose_sign1 BLOB NOT NULL,
  tree_index INTEGER NOT NULL UNIQUE,
  registered_at INTEGER NOT NULL,
  issuer_id TEXT,
  payload_hash BLOB NOT NULL
);
-- ... (full schema from data-model.md)
```

**Acceptance**: Schema applies cleanly to SQLite database

---

### T012: Set up MinIO bucket structure
**Story**: Foundation
**Description**: Document MinIO bucket organization for tiles
**Files**:
- `/docs/minio-setup.md`

**Content**:
```
Buckets:
- scitt-tiles/ (tile storage)
- scitt-artifacts/ (other artifacts)
```

**Acceptance**: Documentation clear for deployment

---

## Phase 3: User Story 1 - Language-Independent Quick Start [P1]

**Goal**: Users can start with either language independently within 5 minutes

**Independent Test Criteria**:
- ✅ Follow TypeScript README → install, init service, create statement (no Go references)
- ✅ Follow Go README → build, init service, create statement (no TypeScript references)
- ✅ CLI help commands show consistent structure

---

### T013 [P]: Migrate existing TypeScript code to scitt-typescript/
**Story**: US1
**Description**: Move src/, tests/, index.ts to scitt-typescript/ with updated imports
**Files**:
- Move `/src/**/*` → `/scitt-typescript/src/`
- Move `/tests/**/*` → `/scitt-typescript/tests/`
- Move `/index.ts` → `/scitt-typescript/src/index.ts`
- Update all import paths (`./src/lib/` → `scitt-typescript/lib/`)

**Script**:
```bash
git mv src scitt-typescript/
git mv tests scitt-typescript/
find scitt-typescript -name "*.ts" -exec sed -i '' 's|from ["'\'']\./src/|from "scitt-typescript/|g' {} \;
```

**Acceptance**: All existing tests pass in new location

---

### T014: Create root README with language navigation
**Story**: US1
**Description**: Root README that helps users choose language (FR-023)
**Files**:
- `/README.md` (replace existing)

**Content** (from quickstart.md):
```markdown
# SCITT Transparency Service

Choose your implementation:

## [TypeScript →](./scitt-typescript/README.md)
Use if you prefer TypeScript/JavaScript, Bun runtime, Node.js integration

## [Go →](./scitt-golang/README.md)
Use if you prefer Go, compiled binaries, minimal dependencies

Both implementations are 100% interoperable.
```

**Acceptance**: No language-specific instructions in root README (FR-011, SC-001)

---

### T015 [P]: Create TypeScript-specific README
**Story**: US1
**Description**: Detailed TypeScript quick start (no Go references)
**Files**:
- `/scitt-typescript/README.md`

**Content**: Extract TypeScript section from quickstart.md
- Installation via install-typescript.sh
- CLI commands with scitt-ts
- Bun-specific instructions
- No mention of Go

**Acceptance**: User completes quick start in <5 minutes without Go knowledge (SC-001)

---

### T016 [P]: Create Go-specific README
**Story**: US1
**Description**: Detailed Go quick start (no TypeScript references)
**Files**:
- `/scitt-golang/README.md`

**Content**: Extract Go section from quickstart.md
- Installation via install-golang.sh
- CLI commands with scitt
- Go-specific instructions
- No mention of TypeScript

**Acceptance**: User completes quick start in <5 minutes without TypeScript knowledge (SC-001)

---

### T017 [P]: Implement TypeScript CLI entry point
**Story**: US1
**Description**: Main CLI with help command structure
**Files**:
- `/scitt-typescript/src/cli/index.ts`
- `/scitt-typescript/src/cli/commands/index.ts`

**Implementation**:
```typescript
#!/usr/bin/env bun
import { program } from 'commander';
program
  .name('scitt-ts')
  .description('SCITT Transparency Service CLI')
  .version('1.0.0');
// ... register commands
program.parse();
```

**Acceptance**: `scitt-ts --help` shows consistent command structure (FR-014)

---

### T018 [P]: Implement Go CLI entry point
**Story**: US1
**Description**: Main CLI with help command structure (matching TypeScript)
**Files**:
- `/scitt-golang/cmd/scitt/main.go`

**Implementation**:
```go
package main

import (
    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:   "scitt",
    Short: "SCITT Transparency Service CLI",
    Version: "1.0.0",
}
// ... register commands
```

**Acceptance**: `scitt --help` matches `scitt-ts --help` structure (FR-014, SC-003)

---

### T019 [P]: Implement CLI keys command (TypeScript)
**Story**: US1
**Description**: scitt-ts keys generate command
**Files**:
- `/scitt-typescript/src/cli/commands/keys.ts`

**Implementation**:
```typescript
program
  .command('keys')
  .command('generate')
  .option('--output <path>', 'Output file path')
  .action(async (options) => {
    const keys = await generateES256Key();
    await writeFile(options.output, JSON.stringify(keys));
  });
```

**Acceptance**: Generates COSE Key file (FR-003)

---

### T020 [P]: Implement CLI keys command (Go)
**Story**: US1
**Description**: scitt keys generate command (matching TypeScript)
**Files**:
- `/scitt-golang/internal/cli/keys.go`

**Implementation**:
```go
var keysCmd = &cobra.Command{
    Use:   "keys",
}

var keysGenerateCmd = &cobra.Command{
    Use:   "generate",
    Run: func(cmd *cobra.Command, args []string) {
        keys, _ := keys.GenerateES256Key()
        // write to output file
    },
}
```

**Acceptance**: Output format identical to TypeScript (FR-014, SC-003)

---

### T021 [P]: Implement CLI identity command (TypeScript)
**Story**: US1
**Description**: scitt-ts identity create command
**Files**:
- `/scitt-typescript/src/cli/commands/identity.ts`

**Acceptance**: Creates .well-known compatible files (FR-004)

---

### T022 [P]: Implement CLI identity command (Go)
**Story**: US1
**Description**: scitt identity create command (matching TypeScript)
**Files**:
- `/scitt-golang/internal/cli/identity.go`

**Acceptance**: Output files identical to TypeScript (FR-004, SC-003)

---

### T023 [P]: Implement CLI statement command (TypeScript)
**Story**: US1
**Description**: scitt-ts statement sign command
**Files**:
- `/scitt-typescript/src/cli/commands/statement.ts`

**Acceptance**: Creates COSE Sign1 statement (FR-005)

---

### T024 [P]: Implement CLI statement command (Go)
**Story**: US1
**Description**: scitt statement sign command (matching TypeScript)
**Files**:
- `/scitt-golang/internal/cli/statement.go`

**Acceptance**: COSE Sign1 format identical to TypeScript (FR-005, SC-003)

---

### T025: Create installation scripts
**Story**: US1
**Description**: Platform detection and binary download scripts
**Files**:
- `/docs/install/install-typescript.sh`
- `/docs/install/install-golang.sh`

**Content**: From quickstart.md, platform detection (macOS, Linux only)

**Acceptance**: Scripts work on macOS and Linux

---

### T026: Test User Story 1 (TypeScript path)
**Story**: US1
**Description**: Execute TypeScript quick start end-to-end
**Steps**:
1. Read `/README.md` → choose TypeScript
2. Run `/docs/install/install-typescript.sh`
3. Generate key: `scitt-ts keys generate`
4. Create identity: `scitt-ts identity create`
5. Sign statement: `scitt-ts statement sign`
6. Measure time: MUST complete in <5 minutes

**Acceptance**: SC-001 verified for TypeScript

---

### T027: Test User Story 1 (Go path)
**Story**: US1
**Description**: Execute Go quick start end-to-end
**Steps**:
1. Read `/README.md` → choose Go
2. Run `/docs/install/install-golang.sh`
3. Generate key: `scitt keys generate`
4. Create identity: `scitt identity create`
5. Sign statement: `scitt statement sign`
6. Measure time: MUST complete in <5 minutes

**Acceptance**: SC-001 verified for Go

---

✅ **CHECKPOINT US1**: Users can independently quick-start in either language

---

## Phase 4: User Story 2 - Cross-Implementation Interoperability [P1]

**Goal**: Artifacts from one implementation work in the other

**Independent Test Criteria**:
- ✅ Generate key in TypeScript → sign in Go → verify successfully
- ✅ Create issuer identity in Go → validate in TypeScript
- ✅ Sign statement in TypeScript → generate receipt in Go → verify
- ✅ Receipts from both implementations validate each other

---

### T028: Set up test vector generation (Go)
**Story**: US2
**Description**: Go program to generate interoperability test vectors
**Files**:
- `/scitt-golang/tests/vectors/generate.go`

**Implementation**:
```go
func GenerateTestVectors() {
    vectors := TestVectorSet{
        Version: "1.0.0",
        Algorithm: "ES256",
        TestCases: []TestCase{
            // key generation, statement signing, etc.
        },
    }
    // Write to tests/interop/fixtures/
}
```

**Acceptance**: Generates JSON test vectors per research.md

---

### T029: Generate ES256 key test vectors (Go)
**Story**: US2
**Description**: Test vectors for key generation
**Files**:
- `/tests/interop/fixtures/keys/es256-keygen.json`

**Content**: 10 test cases with public/private key pairs

**Acceptance**: JSON format matches data-model.md

---

### T030 [P]: Implement COSE Sign1 signing (TypeScript)
**Story**: US2
**Description**: Core COSE Sign1 implementation for statements
**Files**:
- `/scitt-typescript/src/lib/cose/sign1.ts`

**Implementation**:
```typescript
export async function signCoseSign1(
  payload: Uint8Array,
  privateKey: CoseKey
): Promise<Uint8Array> {
  // COSE_Sign1 = [protected, unprotected, payload, signature]
  // alg: ES256 (-7)
}
```

**Acceptance**: Generates valid COSE Sign1 per RFC 8152

---

### T031 [P]: Implement COSE Sign1 signing (Go)
**Story**: US2
**Description**: Core COSE Sign1 implementation (canonical reference)
**Files**:
- `/scitt-golang/pkg/cose/sign1.go`

**Implementation**:
```go
func SignCoseSign1(payload []byte, privateKey *CoseKey) ([]byte, error) {
    // COSE_Sign1 = [protected, unprotected, payload, signature]
    // alg: ES256 (-7)
}
```

**Acceptance**: Canonical reference per Principle VIII

---

### T032: Generate statement signing test vectors (Go)
**Story**: US2
**Description**: COSE Sign1 test vectors
**Files**:
- `/tests/interop/fixtures/statements/cose-sign1.json`

**Content**: 20 test cases with payloads and signed statements

**Acceptance**: TypeScript can verify all signatures

---

### T033: Implement test vector validation (TypeScript)
**Story**: US2
**Description**: Load and validate Go-generated test vectors
**Files**:
- `/scitt-typescript/tests/integration/test-vectors.test.ts`

**Implementation**:
```typescript
describe('Go Test Vectors', () => {
  test('validates ES256 keys', async () => {
    const vectors = loadFixture('keys/es256-keygen.json');
    for (const tc of vectors.testCases) {
      const valid = await validateKey(tc.output.publicKey);
      expect(valid).toBe(true);
    }
  });
});
```

**Acceptance**: All Go-generated vectors validate (SC-002)

---

### T034 [P]: Implement Merkle tree (TypeScript)
**Story**: US2
**Description**: RFC 6962 Merkle tree with C2SP tile format
**Files**:
- `/scitt-typescript/src/lib/merkle/tile-log.ts`
- `/scitt-typescript/src/lib/merkle/proofs.ts`

**Implementation**: Existing code, ensure compliance with Go

**Acceptance**: Generates tiles identical to Go (Principle VIII)

---

### T035 [P]: Implement Merkle tree (Go)
**Story**: US2
**Description**: Use golang.org/x/mod/sumdb/tlog (canonical)
**Files**:
- `/scitt-golang/pkg/merkle/tilelog.go`
- `/scitt-golang/pkg/merkle/proofs.go`

**Implementation**:
```go
import "golang.org/x/mod/sumdb/tlog"

type TileLog struct {
    storage Storage
    tree    tlog.Tree
}
```

**Acceptance**: Canonical reference for TypeScript

---

### T036: Generate tile test vectors (Go)
**Story**: US2
**Description**: C2SP tile format test vectors
**Files**:
- `/tests/interop/fixtures/tiles/full-tiles.json`
- `/tests/interop/fixtures/tiles/partial-tiles.json`

**Content**: Tile data for sizes 127, 256, 512

**Acceptance**: TypeScript generates identical tiles

---

### T037 [P]: Implement receipt generation (TypeScript)
**Story**: US2
**Description**: COSE Merkle proof receipts
**Files**:
- `/scitt-typescript/src/lib/receipt/generate.ts`

**Implementation**:
```typescript
export async function generateReceipt(
  statement: Uint8Array,
  treeIndex: number,
  tileLog: TileLog,
  serviceKey: CoseKey
): Promise<Uint8Array> {
  // COSE Sign1 with inclusion proof
}
```

**Acceptance**: Receipt format per draft-ietf-cose-merkle-tree-proofs-17

---

### T038 [P]: Implement receipt generation (Go)
**Story**: US2
**Description**: COSE Merkle proof receipts (canonical)
**Files**:
- `/scitt-golang/pkg/receipt/generate.go`

**Acceptance**: Canonical reference for TypeScript

---

### T039: Generate receipt test vectors (Go)
**Story**: US2
**Description**: Receipt test vectors with inclusion proofs
**Files**:
- `/tests/interop/fixtures/receipts/cose-proofs.json`

**Content**: 30 test cases with statements and receipts

**Acceptance**: TypeScript verifies all receipts

---

### T040: Cross-test: TypeScript key → Go sign
**Story**: US2
**Description**: Generate key in TypeScript, sign statement in Go
**Files**:
- `/tests/interop/cross-key-sign.test.ts`
- `/tests/interop/cross-key-sign_test.go`

**Steps**:
1. Generate key with scitt-ts
2. Sign statement with scitt (Go)
3. Verify signature in both implementations

**Acceptance**: Signature verifies successfully (SC-009)

---

### T041: Cross-test: Go key → TypeScript sign
**Story**: US2
**Description**: Generate key in Go, sign statement in TypeScript
**Files**:
- `/tests/interop/cross-go-key.test.ts`

**Acceptance**: Signature verifies successfully (SC-009)

---

### T042: Cross-test: Issuer identity exchange
**Story**: US2
**Description**: Create identity in one, validate in other
**Files**:
- `/tests/interop/identity-exchange.test.ts`
- `/tests/interop/identity-exchange_test.go`

**Steps**:
1. Create identity in TypeScript
2. Fetch and validate in Go
3. Reverse: Create in Go, validate in TypeScript

**Acceptance**: Both directions successful (SC-009)

---

### T043: Cross-test: Receipt interoperability
**Story**: US2
**Description**: Receipts from either implementation validate
**Files**:
- `/tests/interop/receipt-interop.test.ts`
- `/tests/interop/receipt-interop_test.go`

**Steps**:
1. Register statement in TypeScript server
2. Get receipt
3. Verify receipt in Go
4. Reverse

**Acceptance**: 100% validation success (SC-002, SC-009)

---

### T044: Test User Story 2 complete
**Story**: US2
**Description**: Execute all interoperability scenarios
**Steps**:
1. Run all cross-implementation tests
2. Verify 100% pass rate

**Acceptance**: SC-002, SC-009 verified

---

✅ **CHECKPOINT US2**: 100% interoperability verified

---

## Phase 5: User Story 3 - Consistent CLI Operations [P2]

**Goal**: CLI commands identical across implementations

**Independent Test Criteria**:
- ✅ Execute identical command sequences in both CLIs
- ✅ Output formats match exactly
- ✅ Error messages consistent

---

### T045: Implement CLI receipt command (TypeScript)
**Story**: US3
**Description**: scitt-ts receipt verify command
**Files**:
- `/scitt-typescript/src/cli/commands/receipt.ts`

**Acceptance**: Verifies receipts against checkpoints (FR-006)

---

### T046: Implement CLI receipt command (Go)
**Story**: US3
**Description**: scitt receipt verify command (matching TypeScript)
**Files**:
- `/scitt-golang/internal/cli/receipt.go`

**Acceptance**: Output format identical to TypeScript (FR-006, SC-003)

---

### T047: Standardize CLI error messages (TypeScript)
**Story**: US3
**Description**: Consistent error format across commands
**Files**:
- `/scitt-typescript/src/cli/errors.ts`

**Implementation**:
```typescript
export function formatError(error: Error): string {
  return `Error: ${error.message}\nDetails: ${error.cause}`;
}
```

**Acceptance**: Error format matches Go (FR-016)

---

### T048: Standardize CLI error messages (Go)
**Story**: US3
**Description**: Consistent error format matching TypeScript
**Files**:
- `/scitt-golang/internal/cli/errors.go`

**Acceptance**: Error format identical to TypeScript (FR-016, SC-003)

---

### T049: Implement CLI help consistency
**Story**: US3
**Description**: Ensure help text matches across implementations
**Files**:
- `/scitt-typescript/src/cli/help.ts`
- `/scitt-golang/internal/cli/help.go`

**Acceptance**: Help text byte-for-byte identical (FR-014)

---

### T050: Add CLI output formatting (TypeScript)
**Story**: US3
**Description**: Consistent output for keys, identities, statements
**Files**:
- `/scitt-typescript/src/cli/format.ts`

**Acceptance**: Output formats documented and consistent

---

### T051: Add CLI output formatting (Go)
**Story**: US3
**Description**: Matching output formats
**Files**:
- `/scitt-golang/internal/cli/format.go`

**Acceptance**: Formats identical to TypeScript (SC-003)

---

### T052: Test CLI command parity
**Story**: US3
**Description**: Automated comparison of CLI outputs
**Files**:
- `/tests/interop/cli-parity.test.ts`

**Steps**:
1. Execute same command in both CLIs
2. Compare stdout, stderr, exit codes
3. Test all commands

**Acceptance**: 100% output match (SC-003)

---

### T053: Test CLI error handling parity
**Story**: US3
**Description**: Verify error messages match
**Files**:
- `/tests/interop/cli-errors.test.ts`

**Steps**:
1. Trigger errors in both CLIs
2. Compare error messages

**Acceptance**: Error messages identical (FR-016)

---

### T054: Document CLI command reference
**Story**: US3
**Description**: Complete CLI documentation
**Files**:
- `/docs/cli-reference.md`

**Content**: All commands with examples, flags, outputs

**Acceptance**: Documentation matches both implementations

---

### T055: Create CLI installation test
**Story**: US3
**Description**: Verify install scripts work correctly
**Files**:
- `/tests/install/test-typescript-install.sh`
- `/tests/install/test-golang-install.sh`

**Acceptance**: Both install scripts succeed on macOS and Linux

---

### T056: Test User Story 3 complete
**Story**: US3
**Description**: Execute all CLI consistency tests
**Acceptance**: SC-003 verified (CLI commands identical)

---

✅ **CHECKPOINT US3**: CLI operations fully consistent

---

## Phase 6: User Story 4 - Server Deployment Parity [P2]

**Goal**: Equivalent server APIs and deployment

**Independent Test Criteria**:
- ✅ Deploy both servers with identical config
- ✅ All API endpoints return structurally equivalent responses
- ✅ Performance targets met by both

---

### T057 [P]: Implement SCRAPI endpoints (TypeScript)
**Story**: US4
**Description**: All SCRAPI mandatory + optional endpoints
**Files**:
- `/scitt-typescript/src/server/api/entries.ts` (POST /entries, GET /entries/{id})
- `/scitt-typescript/src/server/api/wellknown.ts` (.well-known endpoints)
- `/scitt-typescript/src/server/api/issuers.ts` (GET /issuers/{id})

**Acceptance**: All endpoints per openapi.yaml (FR-007)

---

### T058 [P]: Implement SCRAPI endpoints (Go)
**Story**: US4
**Description**: All SCRAPI endpoints matching TypeScript
**Files**:
- `/scitt-golang/internal/server/api/entries.go`
- `/scitt-golang/internal/server/api/wellknown.go`
- `/scitt-golang/internal/server/api/issuers.go`

**Acceptance**: Response schemas identical to TypeScript (FR-007, SC-004)

---

### T059 [P]: Implement C2SP tile endpoints (TypeScript)
**Story**: US4
**Description**: All C2SP tile log endpoints
**Files**:
- `/scitt-typescript/src/server/api/tiles.ts` (GET /tile/{L}/{N}, partials)
- `/scitt-typescript/src/server/api/checkpoint.ts` (GET /checkpoint)

**Acceptance**: Tile format per C2SP spec (FR-007)

---

### T060 [P]: Implement C2SP tile endpoints (Go)
**Story**: US4
**Description**: C2SP endpoints matching TypeScript
**Files**:
- `/scitt-golang/internal/server/api/tiles.go`
- `/scitt-golang/internal/server/api/checkpoint.go`

**Acceptance**: Tile data byte-identical to TypeScript (SC-004)

---

### T061 [P]: Implement media type middleware (TypeScript)
**Story**: US4
**Description**: Enforce correct Content-Type headers
**Files**:
- `/scitt-typescript/src/server/middleware/content-type.ts`

**Acceptance**: Returns 415 for incorrect media types

---

### T062 [P]: Implement media type middleware (Go)
**Story**: US4
**Description**: Matching middleware
**Files**:
- `/scitt-golang/internal/server/middleware/contenttype.go`

**Acceptance**: Behavior identical to TypeScript

---

### T063: Implement SQLite storage adapter (TypeScript)
**Story**: US4
**Description**: Database operations using better-sqlite3
**Files**:
- `/scitt-typescript/src/lib/storage/sqlite.ts`

**Acceptance**: Schema from docs/schema.sql

---

### T064: Implement SQLite storage adapter (Go)
**Story**: US4
**Description**: Database operations using mattn/go-sqlite3
**Files**:
- `/scitt-golang/pkg/storage/sqlite.go`

**Acceptance**: Schema identical to TypeScript

---

### T065: Implement MinIO storage adapter (TypeScript)
**Story**: US4
**Description**: Tile storage in MinIO
**Files**:
- `/scitt-typescript/src/lib/storage/minio.ts`

**Acceptance**: Stores tiles in correct bucket structure

---

### T066: Implement MinIO storage adapter (Go)
**Story**: US4
**Description**: Matching tile storage
**Files**:
- `/scitt-golang/pkg/storage/minio.go`

**Acceptance**: Bucket structure identical to TypeScript

---

### T067: Create server configuration (TypeScript)
**Story**: US4
**Description**: Environment-based configuration
**Files**:
- `/scitt-typescript/src/server/config.ts`

**Acceptance**: Supports all environment variables from quickstart.md (FR-015)

---

### T068: Create server configuration (Go)
**Story**: US4
**Description**: Matching configuration
**Files**:
- `/scitt-golang/internal/server/config.go`

**Acceptance**: Config format identical to TypeScript (FR-015)

---

### T069: Contract test SCRAPI endpoints (TypeScript)
**Story**: US4
**Description**: Validate against openapi.yaml
**Files**:
- `/scitt-typescript/tests/contract/scrapi.test.ts`

**Acceptance**: All endpoints comply with OpenAPI spec

---

### T070: Contract test SCRAPI endpoints (Go)
**Story**: US4
**Description**: Matching contract tests
**Files**:
- `/scitt-golang/tests/contract/scrapi_test.go`

**Acceptance**: Same compliance as TypeScript

---

### T071: Performance test servers
**Story**: US4
**Description**: Verify performance targets (SC-006)
**Files**:
- `/tests/performance/statement-registration.test.ts`

**Tests**:
- Statement registration <200ms p95
- Receipt generation <100ms p95
- Tile retrieval <50ms p95
- Throughput 1000 req/s

**Acceptance**: Both implementations meet targets (SC-006)

---

### T072: Test User Story 4 complete
**Story**: US4
**Description**: Deploy both servers, verify parity
**Steps**:
1. Deploy TypeScript server
2. Deploy Go server
3. Exercise all endpoints
4. Compare responses

**Acceptance**: SC-004, SC-006 verified

---

✅ **CHECKPOINT US4**: Server deployment parity achieved

---

## Phase 7: User Story 5 - Library Integration [P3]

**Goal**: Programmatic library APIs

**Independent Test Criteria**:
- ✅ Import library in minimal code
- ✅ Create, sign, verify statements programmatically
- ✅ APIs parallel across implementations

---

### T073 [P]: Create library public API (TypeScript)
**Story**: US5
**Description**: Export clean library interface
**Files**:
- `/scitt-typescript/src/lib/index.ts`

**Content**:
```typescript
export { generateES256Key } from './keys';
export { signStatement, verifyStatement } from './statement';
export { generateReceipt, verifyReceipt } from './receipt';
// ... all public APIs
```

**Acceptance**: Clean, documented API surface (FR-008)

---

### T074 [P]: Create library public API (Go)
**Story**: US5
**Description**: Parallel Go API
**Files**:
- `/scitt-golang/pkg/scitt/scitt.go`

**Content**: Matching function signatures

**Acceptance**: API surface parallel to TypeScript (FR-008)

---

### T075: Document library usage (TypeScript)
**Story**: US5
**Description**: Library integration examples
**Files**:
- `/scitt-typescript/docs/library-usage.md`

**Content**: Code examples from quickstart.md

**Acceptance**: Examples work as documented

---

### T076: Document library usage (Go)
**Story**: US5
**Description**: Matching documentation
**Files**:
- `/scitt-golang/docs/library-usage.md`

**Acceptance**: Examples parallel to TypeScript

---

### T077: Test library integration (TypeScript)
**Story**: US5
**Description**: Minimal integration test
**Files**:
- `/tests/interop/library-typescript.test.ts`

**Steps**: Execute quickstart.md library example

**Acceptance**: Completes without errors

---

### T078: Test library integration (Go)
**Story**: US5
**Description**: Matching integration test
**Files**:
- `/tests/interop/library-golang_test.go`

**Acceptance**: Output compatible with TypeScript artifacts

---

✅ **CHECKPOINT US5**: Library APIs complete

---

## Phase 8: Polish & Integration

**Goal**: Cross-cutting concerns and final validation

### T079: Update CLAUDE.md
**Story**: Polish
**Description**: Agent context with new structure
**Files**:
- `/CLAUDE.md`

**Content**: Monorepo structure, dual languages, key commands

**Acceptance**: Agent understands project structure

---

### T080: Create CONTRIBUTING.md
**Story**: Polish
**Description**: Contribution guidelines for dual implementation
**Files**:
- `/CONTRIBUTING.md`

**Content**:
- How to contribute to either implementation
- Interoperability test requirements
- Lockstep versioning policy

**Acceptance**: Clear contribution process

---

### T081: Set up CI/CD workflows
**Story**: Polish
**Description**: Complete GitHub Actions configuration
**Files**: Update workflows created in T005
- Path-based triggers
- Interop test gates
- Release coordination

**Acceptance**: CI runs correctly, interop tests block releases (FR-020)

---

### T082: Create release workflow
**Story**: Polish
**Description**: Coordinated lockstep release process
**Files**:
- `/.github/workflows/release.yml`

**Content**:
- Tag both implementations simultaneously
- Build binaries for macOS and Linux
- Publish to GitHub releases

**Acceptance**: Single release creates both binaries (FR-021)

---

### T083: Final interoperability validation
**Story**: Polish
**Description**: Run complete test suite
**Files**:
- Run all tests in `/tests/interop/`

**Acceptance**: 100% pass rate (SC-002)

---

### T084: Performance validation
**Story**: Polish
**Description**: Verify both implementations meet performance targets
**Acceptance**: SC-006 verified

---

### T085: Documentation review
**Story**: Polish
**Description**: Verify all documentation accurate and consistent
**Files**: Review all README files, docs/

**Acceptance**: SC-005 verified (parallel documentation paths)

---

### T086: Test configuration compatibility
**Story**: US4
**Description**: Verify configuration files work identically in both implementations
**Files**:
- `/tests/interop/config-compat.test.ts`
- `/tests/interop/config-compat_test.go`

**Implementation**:
```typescript
// TypeScript test
describe('Configuration Compatibility', () => {
  test('loads same config.json in both implementations', async () => {
    const config = loadConfig('./test-fixtures/config.json');
    expect(config.service.id).toBe('https://transparency.example.com');
    expect(config.server.port).toBe(8080);
    // Verify all config values parse identically
  });
});
```

```go
// Go test
func TestConfigurationCompatibility(t *testing.T) {
    config, err := LoadConfig("./test-fixtures/config.json")
    if err != nil {
        t.Fatal(err)
    }
    assert.Equal(t, "https://transparency.example.com", config.Service.ID)
    assert.Equal(t, 8080, config.Server.Port)
    // Verify all config values parse identically
}
```

**Steps**:
1. Create test config.json with all supported options
2. Load config in TypeScript server
3. Load config in Go server
4. Compare parsed values field-by-field
5. Test environment variable overrides
6. Test error handling for invalid config

**Acceptance**:
- Same config.json works in both implementations (FR-015)
- Environment variables override config identically
- Error messages consistent for invalid config

---

## Dependencies

```
Phase 1 (Setup) → Phase 2 (Foundation) → Phase 3+ (User Stories)

User Stories:
  US1 (P1): Independent (MVP)
  US2 (P1): Requires US1 (needs CLI + basic operations)
  US3 (P2): Requires US1, US2 (needs full CLI + interop)
  US4 (P2): Requires US1, US2 (needs core functionality + interop)
  US5 (P3): Requires US1, US2, US4 (needs CLI, server, interop)
```

## Parallel Execution Opportunities

**Within Phase 1** (Setup):
- T002 [TypeScript init] || T003 [Go init]
- T005 [GitHub Actions] || T007 [TS dirs] || T008 [Go dirs]

**Within Phase 2** (Foundation):
- T009 [TS keys] || T010 [Go keys]

**Within Phase 3** (US1):
- T015 [TS README] || T016 [Go README]
- T017 [TS CLI] || T018 [Go CLI]
- T019 [TS keys cmd] || T020 [Go keys cmd]
- T021 [TS identity cmd] || T022 [Go identity cmd]
- T023 [TS statement cmd] || T024 [Go statement cmd]

**Within Phase 4** (US2):
- T030 [TS COSE] || T031 [Go COSE]
- T034 [TS Merkle] || T035 [Go Merkle]
- T037 [TS receipt] || T038 [Go receipt]

**Within Phase 6** (US4):
- T057 [TS SCRAPI] || T058 [Go SCRAPI]
- T059 [TS tiles] || T060 [Go tiles]
- T061 [TS middleware] || T062 [Go middleware]

## Implementation Strategy

**MVP Scope**: User Story 1 (Tasks T001-T027)
- Users can quick-start in either language independently
- Basic CLI operations (keys, identity, statement)
- No server deployment yet

**Incremental Delivery**:
1. **Sprint 1**: Phase 1-3 (Setup + US1) → MVP
2. **Sprint 2**: Phase 4 (US2) → Interoperability validated
3. **Sprint 3**: Phase 5 (US3) → CLI complete
4. **Sprint 4**: Phase 6 (US4) → Server deployable
5. **Sprint 5**: Phase 7-8 (US5 + Polish) → Production ready

**Testing Strategy**:
- Constitution Principle III requires test-first development
- Cross-implementation tests validate interoperability (US2)
- Each user story has independent test criteria
- Interop tests MUST pass before merge (FR-020)

---

**Total Tasks**: 86
**MVP Tasks**: 27 (T001-T027)
**Parallel Opportunities**: 30+ tasks can run in parallel
**Estimated MVP Delivery**: 2-3 weeks
**Full Implementation**: 5 sprints

**Next Step**: Begin with Phase 1 (Setup) tasks T001-T008
