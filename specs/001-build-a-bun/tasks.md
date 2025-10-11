# Tasks: IETF Standards-Based Transparency Service

**Input**: Design documents from `specs/001-build-a-bun/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Test-First Development (TDD) is **MANDATORY** per project constitution. Tests MUST be written before implementation, verified to fail, then implemented to pass (Red-Green-Refactor).

**Organization**: Tasks are grouped by user story (P1-P4) to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions
- **Single project**: `src/`, `tests/` at repository root
- Bun + TypeScript with native SQLite and built-in test runner

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure needed by all user stories

- [x] T001 Initialize Bun project with package.json and tsconfig.json at repository root
- [x] T002 [P] Create src/ directory structure: lib/, service/, cli/, types/
- [x] T003 [P] Create tests/ directory structure: contract/, integration/, unit/
- [x] T004 [P] Define shared TypeScript types in src/types/cose.ts (COSE Sign1, Hash Envelope structures)
- [x] T005 [P] Define shared TypeScript types in src/types/scitt.ts (SCITT data types)
- [x] T006 [P] Define shared TypeScript types in src/types/config.ts (Configuration types)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Storage Abstraction (Required by all stories)

- [x] T007 Write unit tests for object storage interface in tests/unit/storage/interface.test.ts
- [x] T008 Implement object storage interface in src/lib/storage/interface.ts (put, get, exists, list)
- [x] T009 [P] Write unit tests for local storage in tests/unit/storage/local.test.ts
- [ ] T010 [P] Write unit tests for MinIO storage in tests/unit/storage/minio.test.ts
- [x] T011 [P] Implement local filesystem storage in src/lib/storage/local.ts
- [ ] T012 [P] Implement MinIO/S3-compatible storage in src/lib/storage/minio.ts
- [ ] T013 [P] Implement Azure Blob storage in src/lib/storage/azure.ts
- [ ] T014 [P] Implement S3 storage in src/lib/storage/s3.ts

### Database Layer (Required by all stories)

- [x] T015 Write unit tests for database schema in tests/unit/database/schema.test.ts
- [x] T016 Implement SQLite schema and migrations in src/lib/database/schema.ts (statements, receipts, tiles, tree_state)
- [ ] T017 [P] Write unit tests for statements queries in tests/unit/database/statements.test.ts
- [ ] T018 [P] Write unit tests for receipts storage in tests/unit/database/receipts.test.ts
- [ ] T019 [P] Write unit tests for log state in tests/unit/database/log-state.test.ts
- [ ] T020 [P] Implement statement metadata queries in src/lib/database/statements.ts
- [ ] T021 [P] Implement receipt storage/retrieval in src/lib/database/receipts.ts
- [ ] T022 [P] Implement Merkle tree state management in src/lib/database/log-state.ts

### COSE Cryptography (Required by all stories)

**Sanity Tests & Core Primitives** (build from bottom up):

- [x] T023 Write unit tests for COSE key generation (ES256) in tests/unit/cose/key-material.test.ts
- [x] T024 Implement COSE key generation (ES256, P-256) using Web Crypto API in src/lib/cose/key-material.ts
- [x] T025 Write unit tests for COSE key thumbprint computation in tests/unit/cose/key-material.test.ts (extend)
- [x] T026 Implement COSE key thumbprint (RFC 7638 style) in src/lib/cose/key-material.ts (extend)
- [ ] T027 Write unit tests for raw signature creation/verification (Signer/Verifier pattern) in tests/unit/cose/signer.test.ts
- [ ] T028 Implement Signer/Verifier abstraction in src/lib/cose/signer.ts (supports HSM in future, uses Web Crypto for now)

**COSE Sign1 (uses Signer/Verifier)**:

- [ ] T029 Write unit tests for COSE Sign1 signing in tests/unit/cose/sign.test.ts
- [ ] T030 Write unit tests for COSE Sign1 verification in tests/unit/cose/sign.test.ts (extend)
- [ ] T031 Implement COSE Sign1 signing using Signer abstraction in src/lib/cose/sign.ts
- [ ] T032 Implement COSE Sign1 verification using Verifier abstraction in src/lib/cose/sign.ts (extend)

**Issuer Resolution (single choke point for key discovery)**:

- [ ] T033 Write unit tests for issuer resolution (URL → public key) in tests/unit/cose/issuer-resolver.test.ts
- [ ] T034 Write unit tests for kid resolution (kid → key from issuer) in tests/unit/cose/issuer-resolver.test.ts (extend)
- [ ] T035 Implement issuer resolver (fetch from .well-known/*, cache, validate) in src/lib/cose/issuer-resolver.ts
- [ ] T036 Implement kid resolver (resolve specific key by kid) in src/lib/cose/issuer-resolver.ts (extend)

**COSE Hash Envelope** (uses COSE Sign1):

- [ ] T037 Write unit tests for COSE Hash Envelope creation in tests/unit/cose/hash-envelope.test.ts
- [ ] T038 Write unit tests for COSE Hash Envelope validation in tests/unit/cose/hash-envelope.test.ts (extend)
- [ ] T039 Implement COSE Hash Envelope creation (labels 258, 259, 260) in src/lib/cose/hash-envelope.ts
- [ ] T040 Implement COSE Hash Envelope validation in src/lib/cose/hash-envelope.ts (extend)

### Merkle Tree Tile Log (Required by all stories)

**Sanity Tests & Core Primitives** (test in isolation before integration):

- [x] T041 Write unit tests for tile naming (C2SP format: tile/<L>/<N>[.p/<W>]) in tests/unit/merkle/tile-naming.test.ts
- [x] T042 Implement tile naming utilities in src/lib/merkle/tile-naming.ts (index → path, path → index)
- [x] T043 Write unit tests for single leaf append in tests/unit/merkle/tile-log.test.ts
- [x] T044 Write unit tests for full tile creation (256 entries) in tests/unit/merkle/tile-log.test.ts (extend)
- [x] T045 Write unit tests for partial tile handling in tests/unit/merkle/tile-log.test.ts (extend)
- [x] T046 Implement tile-based Merkle tree with C2SP tlog-tiles naming in src/lib/merkle/tile-log.ts

**Proof Generation & Verification** (test independently):

- [ ] T047 Write unit tests for inclusion proof generation in tests/unit/merkle/proofs.test.ts
- [ ] T048 Write unit tests for inclusion proof verification in tests/unit/merkle/proofs.test.ts (extend)
- [ ] T049 Write unit tests for consistency proof generation in tests/unit/merkle/proofs.test.ts (extend)
- [ ] T050 Write unit tests for consistency proof verification in tests/unit/merkle/proofs.test.ts (extend)
- [ ] T051 Implement inclusion and consistency proof generation/verification in src/lib/merkle/proofs.ts

**Checkpoints** (signed tree heads):

- [ ] T052 Write unit tests for checkpoint creation in tests/unit/merkle/checkpoints.test.ts
- [ ] T053 Write unit tests for checkpoint verification in tests/unit/merkle/checkpoints.test.ts (extend)
- [ ] T054 Implement signed tree head (checkpoint) management in src/lib/merkle/checkpoints.ts

**Conformance with transmute-industries/cose**:

- [ ] T055 Port test vectors from transmute-industries/cose COSE tests to tests/unit/cose/conformance.test.ts
- [ ] T056 Port test vectors from transmute-industries/cose Merkle tree tests to tests/unit/merkle/conformance.test.ts
- [ ] T057 Verify conformance: All ported tests pass with our implementation

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Service Operator Deploys Transparency Service (Priority: P1) 🎯 MVP

**Goal**: Service operator can initialize and start a transparency service with database and object storage configuration

**Independent Test**: Deploy service, verify it starts successfully, responds to health checks, and reports healthy status for all components

### Tests for User Story 1 (TDD - Write tests FIRST, verify they FAIL)

- [ ] T035 [P] [US1] Write contract test for transparency configuration endpoint in tests/contract/config.test.ts
- [ ] T036 [P] [US1] Write contract test for service keys endpoint in tests/contract/service-keys.test.ts
- [ ] T037 [P] [US1] Write contract test for health check endpoint in tests/contract/health.test.ts
- [ ] T038 [US1] Write integration test for service initialization workflow in tests/integration/service-init.test.ts

### Implementation for User Story 1

- [ ] T039 [P] [US1] Implement transparency init CLI command in src/cli/commands/transparency/init.ts (create DB, generate keys, configure storage)
- [ ] T040 [P] [US1] Implement CLI utilities for configuration in src/cli/utils/config.ts
- [ ] T041 [P] [US1] Implement CLI output formatting in src/cli/utils/output.ts
- [ ] T042 [US1] Implement HTTP server setup in src/service/server.ts (Bun HTTP server)
- [ ] T043 [P] [US1] Implement transparency configuration endpoint in src/service/routes/config.ts (/.well-known/scitt-configuration)
- [ ] T044 [P] [US1] Implement service keys endpoint in src/service/routes/config.ts (/.well-known/scitt-keys)
- [ ] T045 [P] [US1] Implement health check endpoint in src/service/routes/health.ts
- [ ] T046 [P] [US1] Implement structured logging middleware in src/service/middleware/logging.ts
- [ ] T047 [US1] Implement transparency serve CLI command in src/cli/commands/transparency/serve.ts (start HTTP server)
- [ ] T048 [US1] Implement CLI entry point with command routing in src/cli/index.ts

**Checkpoint**: At this point, User Story 1 is fully functional - service can be deployed, started, and health-checked independently

---

## Phase 4: User Story 2 - Issuer Registers Large Artifact by Hash (Priority: P2)

**Goal**: Issuer can generate identity, sign artifacts as hash envelopes, and register statements with the transparency service

**Independent Test**: Compute hash of large file (1GB parquet), create signed hash envelope, register with service, receive receipt

### Tests for User Story 2 (TDD - Write tests FIRST, verify they FAIL)

- [ ] T049 [P] [US2] Write unit test for streaming hash computation in tests/unit/cose/hash-envelope.test.ts (extend existing)
- [ ] T050 [P] [US2] Write contract test for statement registration endpoint in tests/contract/registration.test.ts
- [ ] T051 [P] [US2] Write contract test for registration status polling in tests/contract/registration.test.ts (extend)
- [ ] T052 [US2] Write integration test for issuer workflow (init → sign → register) in tests/integration/issuer-workflow.test.ts

### Implementation for User Story 2

- [ ] T053 [US2] Implement streaming hash computation for large files in src/lib/cose/hash-envelope.ts (extend, use Bun streaming I/O)
- [ ] T054 [US2] Implement issuer init CLI command in src/cli/commands/issuer/init.ts (generate ES256 key pair, create .well-known materials)
- [ ] T055 [US2] Implement statement sign CLI command in src/cli/commands/statement/sign.ts (stream hash, create hash envelope, sign with COSE Sign1)
- [ ] T056 [P] [US2] Implement SCRAPI types for registration in src/service/types/scrapi.ts
- [ ] T057 [P] [US2] Implement request validation middleware in src/service/middleware/validation.ts
- [ ] T058 [US2] Implement statement registration endpoint in src/service/routes/register.ts (POST /entries, validate, add to tree, generate receipt)
- [ ] T059 [US2] Implement tile creation and storage logic in src/lib/merkle/tile-log.ts (extend, create entry tiles and hash tiles)
- [ ] T060 [US2] Implement receipt generation with inclusion proofs in src/lib/merkle/proofs.ts (extend)
- [ ] T061 [US2] Implement statement register CLI command in src/cli/commands/statement/register.ts (POST to service, handle sync/async, poll status, save receipt)

**Checkpoint**: At this point, User Stories 1 AND 2 both work independently - issuers can register signed statements

---

## Phase 5: User Story 3 - Verifier Checks Statement Authenticity (Priority: P3)

**Goal**: Verifier can verify artifact hash, statement signature, and receipt inclusion proof (offline capable)

**Independent Test**: Receive artifact + transparent statement (hash envelope + receipt), verify hash matches, signature valid, receipt proves inclusion

### Tests for User Story 3 (TDD - Write tests FIRST, verify they FAIL)

- [ ] T062 [P] [US3] Write contract test for receipt resolution endpoint in tests/contract/receipts.test.ts
- [ ] T063 [P] [US3] Write contract test for tile retrieval endpoints in tests/contract/tiles.test.ts
- [ ] T064 [US3] Write integration test for verifier workflow (verify signature → receipt) in tests/integration/verifier-workflow.test.ts
- [ ] T065 [US3] Write integration test for offline verification in tests/integration/verifier-workflow.test.ts (extend, no network)

### Implementation for User Story 3

- [ ] T066 [P] [US3] Implement receipt resolution endpoint in src/service/routes/receipts.ts (GET /entries/{entry-id}/receipt)
- [ ] T067 [P] [US3] Implement tile retrieval endpoints in src/service/routes/tiles.ts (GET /tile/{L}/{N}, GET /tile/entries/{N})
- [ ] T068 [P] [US3] Implement checkpoint endpoint in src/service/routes/checkpoint.ts (GET /checkpoint)
- [ ] T069 [US3] Implement statement verify CLI command in src/cli/commands/statement/verify.ts (compute hash, verify signature, fetch issuer key from URL)
- [ ] T070 [US3] Implement receipt verify CLI command in src/cli/commands/receipt/verify.ts (verify hash → signature → receipt inclusion proof → checkpoint)
- [ ] T071 [US3] Implement offline key caching in src/lib/cose/key-material.ts (extend, local cache for issuer/service keys)

**Checkpoint**: At this point, User Stories 1, 2, AND 3 all work independently - verifiers can verify statements offline

---

## Phase 6: User Story 4 - Auditor Reviews Service Log Consistency (Priority: P4)

**Goal**: Auditor can query log by metadata, verify consistency between checkpoints, and validate data integrity

**Independent Test**: Register multiple statements, verify Merkle tree consistency, ensure append-only property, match SQL/storage

### Tests for User Story 4 (TDD - Write tests FIRST, verify they FAIL)

- [ ] T072 [US4] Write integration test for auditor workflow (consistency checks) in tests/integration/auditor-workflow.test.ts
- [ ] T073 [US4] Write integration test for storage integrity validation in tests/integration/storage-portability.test.ts

### Implementation for User Story 4

- [ ] T074 [US4] Implement log query CLI command in src/cli/commands/transparency/query.ts (query by iss, sub, cty, typ, timestamp)
- [ ] T075 [US4] Implement consistency verification in src/lib/merkle/proofs.ts (extend, verify consistency proofs between tree sizes)
- [ ] T076 [US4] Implement CLI command to verify log consistency in src/cli/commands/transparency/verify-consistency.ts
- [ ] T077 [US4] Implement CLI command to check SQL/storage integrity in src/cli/commands/transparency/check-integrity.ts

**Checkpoint**: All user stories now independently functional - complete transparency service with deployment, registration, verification, and auditing

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T078 [P] Add comprehensive error handling with actionable messages across all CLI commands
- [ ] T079 [P] Add performance logging and metrics collection in src/service/middleware/logging.ts (extend)
- [ ] T080 [P] Create README.md with quickstart guide and architecture overview
- [ ] T081 [P] Create CONTRIBUTING.md with development workflow and testing guidelines
- [ ] T082 [US1] Run quickstart.md validation (manual test of documented workflows)
- [ ] T083 Run full integration test suite to verify all user stories work together
- [ ] T084 Performance testing: Verify SC-002 (1GB file in <30s), SC-003 (10MB in <5s), SC-008 (100 concurrent registrations)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational phase completion
- **User Story 2 (Phase 4)**: Depends on Foundational phase completion (can run parallel with US1 if staffed)
- **User Story 3 (Phase 5)**: Depends on Foundational + US2 (needs registration to test verification)
- **User Story 4 (Phase 6)**: Depends on Foundational + US2 (needs registered statements to audit)
- **Polish (Phase 7)**: Depends on all desired user stories being complete

### User Story Dependencies

- **US1 (P1)**: Can start after Foundational - No dependencies on other stories
- **US2 (P2)**: Can start after Foundational - Depends on US1 for running service (but can be tested with mock)
- **US3 (P3)**: Depends on US2 for registered statements to verify
- **US4 (P4)**: Depends on US2 for registered statements to audit

### Within Each User Story (TDD Order)

1. **Tests written FIRST** (must fail before implementation)
2. **Implementation to pass tests**
3. **Refactor** if needed
4. **Story complete** before moving to next priority

### Parallel Opportunities

**Phase 1: Setup**
- All tasks (T001-T006) can run sequentially but are quick

**Phase 2: Foundational**
- Storage implementations (T009-T014): All [P] after interface tests
- Database implementations (T017-T022): All [P] after schema
- COSE tests can run parallel with implementations once tests pass

**Phase 3: User Story 1**
- Tests (T035-T037): All [P] (different contracts)
- CLI utilities (T040-T041): Both [P]
- Service routes (T043-T045): All [P]

**Phase 4: User Story 2**
- Tests (T049-T051): All [P]
- SCRAPI types + validation (T056-T057): Both [P]

**Phase 5: User Story 3**
- Tests (T062-T063): Both [P]
- Service endpoints (T066-T068): All [P]

**Phase 7: Polish**
- Documentation tasks (T078-T081): All [P]

---

## Parallel Example: User Story 1

```bash
# After Foundational phase completes, launch all US1 tests together:
bun test tests/contract/config.test.ts &
bun test tests/contract/service-keys.test.ts &
bun test tests/contract/health.test.ts &
wait

# Verify tests FAIL (Red phase of TDD)

# Then implement CLI utilities in parallel:
# T040: src/cli/utils/config.ts
# T041: src/cli/utils/output.ts

# Then implement service routes in parallel:
# T043: src/service/routes/config.ts (configuration)
# T044: src/service/routes/config.ts (keys)
# T045: src/service/routes/health.ts
# T046: src/service/middleware/logging.ts
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test US1 independently (deploy service, verify health)
5. Deploy/demo basic transparency service

### Incremental Delivery

1. Foundation (Setup + Foundational) → Ready for user stories
2. Add User Story 1 → Test independently → **MVP deployed!**
3. Add User Story 2 → Test independently → Issuers can register statements
4. Add User Story 3 → Test independently → Verifiers can verify statements
5. Add User Story 4 → Test independently → Auditors can audit log
6. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers after Foundational phase:
- **Developer A**: User Story 1 (service deployment)
- **Developer B**: User Story 2 (statement registration)
- **Developer C**: Storage abstraction + COSE porting (Foundational work)

Once Foundational completes, US1 and US2 can proceed in parallel.

---

## Test-First Development (TDD) Compliance

**Constitution Requirement**: Tests MUST be written before implementation (NON-NEGOTIABLE)

### TDD Workflow for Each Task

1. **Write Test**: Create test file, write test cases based on acceptance criteria
2. **Run Test**: Execute test, verify it FAILS (Red phase)
3. **Implement**: Write minimum code to make test pass
4. **Run Test**: Execute test, verify it PASSES (Green phase)
5. **Refactor**: Improve code quality while keeping tests green
6. **Commit**: Commit passing test + implementation together

### Test Coverage Requirements

- **Unit Tests**: All lib/ components (COSE, Merkle, storage, database)
- **Contract Tests**: All SCRAPI endpoints (compliance with OpenAPI spec)
- **Integration Tests**: Each user story workflow end-to-end
- **Performance Tests**: Success criteria validation (SC-002, SC-003, SC-008)

### Running Tests

```bash
# Run all tests
bun test

# Run specific test file
bun test tests/unit/cose/sign.test.ts

# Run tests for a user story
bun test tests/integration/issuer-workflow.test.ts

# Run contract tests (SCRAPI compliance)
bun test tests/contract/
```

---

## Notes

- **[P] tasks**: Different files, no dependencies - can run in parallel
- **[Story] labels**: Map tasks to user stories for traceability
- **TDD order**: Tests always before implementation (enforced by constitution)
- **Each user story is independently testable**: Can deliver incrementally
- **Foundational phase is critical**: Must complete before any user story
- **Stop at any checkpoint**: Validate story independently before continuing
- **Commit after each Green phase**: Passing test + implementation together
- **File paths are exact**: Ready for LLM execution without additional context

---

## Task Summary

**Total Tasks**: 96
- **Setup**: 6 tasks
- **Foundational**: 40 tasks (BLOCKING) - includes 17 sanity test tasks for COSE + Merkle tree components
- **User Story 1**: 14 tasks (tests: 4, implementation: 10)
- **User Story 2**: 13 tasks (tests: 4, implementation: 9)
- **User Story 3**: 10 tasks (tests: 4, implementation: 6)
- **User Story 4**: 6 tasks (tests: 2, implementation: 4)
- **Polish**: 7 tasks

**Parallel Opportunities**: 32 tasks marked [P]

**Independent Test Points**: 4 (one per user story)

**Suggested MVP Scope**: Phase 1 + Phase 2 + Phase 3 (User Story 1 only) = 60 tasks

**Test-First Ratio**: 26 test tasks : 70 implementation tasks (1:2.7 ratio, TDD enforced with bottom-up sanity tests)
