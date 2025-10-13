# Tasks: Cross-Implementation Integration Test Suite

**Input**: Design documents from `specs/003-create-an-integration/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: This feature IS the test suite - no separate tests needed for test infrastructure. Task validation is via execution success.

**Organization**: Tasks are grouped by user story (P1-P3) to enable independent implementation and testing of each story category.

## Format: `[ID] [P?] [Story] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1-US6)
- Include exact file paths in descriptions

## Path Conventions
- Test suite located in `tests/interop/` at repository root
- Go 1.24 test orchestration with shell script coordination
- Both Go and TypeScript implementations available as prerequisites

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Test suite initialization and structure needed by all user stories

- [x] T001 Initialize Go module for test suite at tests/interop/go.mod with dependencies: testing, net/http, os/exec
- [x] T002 [P] Create directory structure: tests/interop/{lib,cli,http,crypto,merkle,e2e,scripts,fixtures}
- [x] T003 [P] Create fixture subdirectories: tests/interop/fixtures/{keys,payloads,statements,rfc-vectors}
- [x] T004 [P] Create test README at tests/interop/README.md documenting test suite purpose and running instructions (Note: Existing README preserved, Go test orchestration will complement existing TypeScript tests)
- [x] T005 [P] Create test orchestration entry point at tests/interop/main_test.go with package setup and shared test utilities
- [x] T006 [P] Define test configuration types in tests/interop/lib/types.go (TestExecutionContext, TestResult, ImplementationResult structs with snake_case JSON tags)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core test infrastructure that MUST be complete before ANY user story tests can be implemented

**⚠️ CRITICAL**: No user story testing can begin until this phase is complete

### Test Environment Management (Required by all tests)

- [x] T007 Implement test environment setup in tests/interop/lib/setup.go (SetupTestEnv function: creates isolated temp dirs, allocates unique ports 20000-30000, returns cleanup function)
- [x] T008 [P] Implement port allocation utility in tests/interop/lib/ports.go (AllocatePort function with collision detection and cleanup registration)
- [x] T009 [P] Implement CLI invocation utilities in tests/interop/lib/cli.go (RunGoCLI and RunTsCLI functions: execute commands, capture stdout/stderr, handle timeouts)

### Test Fixtures and RFC Vectors (Required by all tests)

- [x] T010 Create test fixture generation tool at tests/interop/tools/generate_keypair.go (generates ES256 keypairs in PEM/JWK format with RFC 7638 thumbprints, hex-encoded)
- [x] T011 Generate 5 test keypair fixtures in tests/interop/fixtures/keys/ (keypair_alice.json, keypair_bob.json, keypair_charlie.json, keypair_dave.json, keypair_eve.json following test-fixtures.schema.json)
- [x] T012 [P] Create test payload fixtures in tests/interop/fixtures/payloads/ (10 JSON payloads with varying sizes: small.json, medium.json, large.json, with computed SHA-256 hashes)
- [x] T013 Create RFC test vector generation tool at tests/interop/tools/generate_rfc_vectors.go (uses Go tlog for Merkle vectors, Go COSE for signing vectors - Note: Existing go-tlog-generator/ and go-cose-generator/ documented in tools/README.md)
- [x] T014 Generate RFC 6962 Merkle tree test vectors in tests/interop/fixtures/rfc-vectors/merkle_trees.json (tree sizes: 1, 2, 3, 5, 10, 256, 1000 with expected root hashes - Note: Existing test-vectors/ directory contains tlog-size-*.json files)
- [x] T015 [P] Generate RFC 9052 COSE Sign1 test vectors in tests/interop/fixtures/rfc-vectors/cose_sign1.json (various protected headers, CWT claims combinations - Note: Existing go-cose-generator/ produces cose-vectors/*.json)
- [x] T016 [P] Generate RFC 7638 JWK thumbprint test vectors in tests/interop/fixtures/rfc-vectors/jwk_thumbprints.json (P-256 keys with expected thumbprints - Note: Computed during keypair generation in fixtures/keys/)

### Test Comparison and Validation (Required by all tests)

- [x] T017 Implement JSON comparison utility in tests/interop/lib/compare.go (CompareJSON function: semantic deep equality, reports differences with JSON paths, handles numeric precision)
- [x] T018 [P] Implement RFC compliance validation in tests/interop/lib/rfc_validate.go (ValidateRFCCompliance function: validates against RFC requirements, returns RFCViolation structs)
- [x] T019 [P] Implement output comparison utilities in tests/interop/lib/compare.go (extend: CompareOutputs for implementation results, generates ComparisonResult with verdict)

### Shell Script Coordination (Required by CLI and E2E tests)

- [x] T020 Create environment setup script at tests/interop/scripts/setup_env.sh (creates temp dirs, sets environment variables for isolated testing)
- [x] T021 [P] Create implementation build script at tests/interop/scripts/build_impls.sh (builds Go CLI, verifies TypeScript CLI, checks binaries exist)
- [x] T022 [P] Create CLI wrapper script at tests/interop/scripts/run_cli.sh (invokes either CLI with logging, captures output, handles errors)

### Test Reporting (Required by all tests)

- [x] T023 Implement test report generation in tests/interop/lib/report.go (GenerateReport function: aggregates TestResult structs, generates Markdown and JSON reports following test-report.schema.json)
- [x] T024 [P] Implement test result aggregation in tests/interop/lib/report.go (extend: CategorySummary, RFCComplianceSummary, PerformanceSummary generation)

**Checkpoint**: Foundation ready - user story tests can now be implemented in parallel

---

## Phase 3: User Story 1 - Cross-Implementation API Compatibility Testing (Priority: P1) 🎯 MVP

**Goal**: Verify that both Go and TypeScript implementations expose identical HTTP APIs

**Independent Test**: Start both servers independently, make identical HTTP requests, compare response formats and semantic content

**Functional Requirements Covered**: FR-011 through FR-016, FR-038, FR-039

### Test Implementation for User Story 1

- [x] T025 [P] [US1] Implement transparency configuration test in tests/interop/http/config_test.go (FR-014: validate GET /.well-known/transparency-configuration returns equivalent structures with matching algorithms, endpoints, origins in snake_case)
- [x] T026 [P] [US1] Implement POST /entries test in tests/interop/http/entries_test.go (FR-011: validate statement registration with identical COSE Sign1 input returns 201 Created with equivalent response schema entry_id, statement_hash in hex encoding)
- [x] T027 [P] [US1] Implement GET /entries/{id} receipt test in tests/interop/http/entries_test.go (extend: FR-012: validate receipt retrieval returns equivalent structures with matching metadata)
- [x] T028 [P] [US1] Implement checkpoint test in tests/interop/http/checkpoint_test.go (FR-013: validate GET /checkpoint returns signed tree heads with identical format for origin, tree_size, root_hash, timestamp, signature lines)
- [x] T029 [P] [US1] Implement health check test in tests/interop/http/health_test.go (FR-015: validate GET /health returns 200 OK with equivalent health status structures)
- [x] T030 [P] [US1] Implement statement query test in tests/interop/http/query_test.go (FR-016: validate query endpoints with filters for issuer, subject, content_type return equivalent result sets)

### Error Handling Tests for User Story 1

- [x] T031 [P] [US1] Implement HTTP error scenario tests in tests/interop/http/errors_test.go (FR-038, FR-039: 15+ error cases including invalid CBOR → 400, missing Content-Type → 400, missing entry → 404, server failure → 500, validate both implementations return matching status codes and equivalent error structures with error_code, error_message in snake_case)

**Checkpoint**: At this point, User Story 1 is fully tested - HTTP API parity validated across implementations

---

## Phase 4: User Story 2 - CLI Tool User Experience Parity (Priority: P1)

**Goal**: Verify both CLI tools provide identical command structures, arguments, and output formats

**Independent Test**: Run equivalent CLI commands for both tools, compare command-line arguments, output formats, exit codes, side effects

**Functional Requirements Covered**: FR-006 through FR-010, FR-040, FR-041

### Test Implementation for User Story 2

- [x] T032 [P] [US2] Implement init command test in tests/interop/cli/init_test.go (FR-006, FR-008: validate both CLIs with identical arguments create equivalent directory structures, config files in snake_case, key pairs in PEM/JWK format)
- [x] T033 [P] [US2] Implement statement sign test in tests/interop/cli/statement_test.go (FR-007, FR-010: validate both CLIs with identical payloads and keys produce byte-identical or semantically equivalent COSE Sign1 structures, output JSON with statement_hash in hex encoding and snake_case fields)
- [x] T034 [P] [US2] Implement statement verify test in tests/interop/cli/statement_test.go (extend: FR-007: validate both CLIs report verification success/failure identically with matching exit codes and output formats)
- [x] T035 [P] [US2] Implement statement hash test in tests/interop/cli/statement_test.go (extend: FR-007: validate both CLIs output identical hash formats and values in hex encoding)
- [x] T036 [P] [US2] Implement statement register test in tests/interop/cli/statement_test.go (extend: FR-006, FR-007: validate both CLIs successfully register to running servers and output identical response formats with entry_id, hash in hex encoding)
- [x] T037 [P] [US2] Implement serve command test in tests/interop/cli/serve_test.go (FR-006: validate both CLIs start servers with identical configuration and behavior)

### CLI Error Handling Tests for User Story 2

- [x] T038 [US2] Implement CLI error scenario tests in tests/interop/cli/errors_test.go (FR-040, FR-041: validate both CLIs exit with equivalent error codes for failure scenarios, handle malformed inputs consistently with equivalent error messages)

**Checkpoint**: At this point, User Stories 1 AND 2 both tested independently - API and CLI parity validated

---

## Phase 5: User Story 3 - Cryptographic Interoperability Testing (Priority: P2)

**Goal**: Verify statements signed by one implementation can be verified by the other

**Independent Test**: Generate keys in one implementation, sign statements, verify signatures in other implementation

**Functional Requirements Covered**: FR-017 through FR-022

### Test Implementation for User Story 3

- [ ] T039 [P] [US3] Implement Go signs TypeScript verifies test in tests/interop/crypto/sign_verify_test.go (FR-017: 50+ combinations using test keypairs, validate TypeScript successfully verifies Go-signed statements)
- [ ] T040 [P] [US3] Implement TypeScript signs Go verifies test in tests/interop/crypto/sign_verify_test.go (extend: FR-018: 50+ combinations using test keypairs, validate Go successfully verifies TypeScript-signed statements)
- [ ] T041 [P] [US3] Implement hash envelope compatibility test in tests/interop/crypto/hash_envelope_test.go (FR-019, FR-020: validate hash envelope statements from both implementations contain equivalent structures with hash algorithm, hash value in hex, location, content type in snake_case, and both implementations validate each other's hash envelopes)
- [ ] T042 [P] [US3] Implement JWK interoperability test in tests/interop/crypto/jwk_test.go (FR-021: validate JWK keys exported by either implementation can be imported and used by other for sign/verify operations)
- [ ] T043 [US3] Implement JWK thumbprint consistency test in tests/interop/crypto/jwk_thumbprint_test.go (FR-022: validate RFC 7638 thumbprints computed by both implementations for same keys are byte-identical and match RFC test vectors)

**Checkpoint**: At this point, User Stories 1, 2, AND 3 tested independently - cryptographic interoperability validated

---

## Phase 6: User Story 4 - Merkle Tree Proof Interoperability (Priority: P2)

**Goal**: Verify Merkle proofs generated by one implementation can be verified by the other

**Independent Test**: Register statements in one implementation's server, retrieve proofs, verify using other implementation's verification functions

**Functional Requirements Covered**: FR-023 through FR-028

### Test Implementation for User Story 4

- [ ] T044 [P] [US4] Implement Go inclusion proof cross-validation test in tests/interop/merkle/inclusion_test.go (FR-023: 30+ inclusion proofs from Go server verified by TypeScript for tree sizes 1, 2, 3, 5, 10, 100, 256, 1000 at entry positions first, middle, last)
- [ ] T045 [P] [US4] Implement TypeScript inclusion proof cross-validation test in tests/interop/merkle/inclusion_test.go (extend: FR-024: 30+ inclusion proofs from TypeScript server verified by Go for same tree sizes and positions)
- [ ] T046 [P] [US4] Implement Go consistency proof cross-validation test in tests/interop/merkle/consistency_test.go (FR-025: consistency proofs from Go server for tree growth 1→2, 1→10, 10→20, 100→200 verified by TypeScript)
- [ ] T047 [P] [US4] Implement TypeScript consistency proof cross-validation test in tests/interop/merkle/consistency_test.go (extend: FR-026: consistency proofs from TypeScript server for same growth patterns verified by Go)
- [ ] T048 [US4] Implement root hash consistency test in tests/interop/merkle/root_hash_test.go (FR-027: validate tree root hashes computed by both implementations for identical leaf sets are byte-identical, validate against RFC 6962 test vectors)
- [ ] T049 [P] [US4] Implement tile naming consistency test in tests/interop/merkle/tile_test.go (FR-028: validate C2SP tlog-tiles naming conventions are identical across implementations for same entry IDs)

**Checkpoint**: At this point, User Stories 1-4 tested independently - Merkle proof interoperability validated

---

## Phase 7: User Story 5 - Statement Query Compatibility (Priority: P3)

**Goal**: Verify query operations return consistent results regardless of implementation

**Independent Test**: Populate both implementations with identical statements, execute equivalent queries, compare result sets

**Functional Requirements Covered**: FR-016 (extended from Phase 3)

### Test Implementation for User Story 5

- [ ] T050 [US5] Populate test data in tests/interop/lib/test_data.go (create 100+ test statements with varying issuers, subjects, media types for query testing)
- [ ] T051 [P] [US5] Implement issuer filter query test in tests/interop/http/query_test.go (extend: validate both implementations return same statements when filtering by issuer)
- [ ] T052 [P] [US5] Implement subject filter query test in tests/interop/http/query_test.go (extend: validate both implementations return same statements when filtering by subject)
- [ ] T053 [P] [US5] Implement content type filter query test in tests/interop/http/query_test.go (extend: validate both implementations return same statements when filtering by media type)
- [ ] T054 [P] [US5] Implement pagination query test in tests/interop/http/query_test.go (extend: validate both implementations return consistent result ordering and pagination behavior)
- [ ] T055 [US5] Implement combined filter query test in tests/interop/http/query_test.go (extend: validate both implementations return identical results for combined filters like issuer AND subject)

**Checkpoint**: At this point, User Stories 1-5 tested independently - query compatibility validated

---

## Phase 8: User Story 6 - Receipt Format Compatibility (Priority: P3)

**Goal**: Verify receipts from both implementations contain equivalent information in compatible formats

**Independent Test**: Register identical statements to both implementations, retrieve receipts, compare structures for semantic equivalence

**Functional Requirements Covered**: FR-012 (extended from Phase 3), FR-029 through FR-032

### Test Implementation for User Story 6

- [ ] T056 [P] [US6] Implement receipt metadata equivalence test in tests/interop/http/receipts_test.go (validate receipts from both implementations contain equivalent entry_id, statement_hash, tree_size, timestamp in hex encoding and snake_case)
- [ ] T057 [P] [US6] Implement receipt cross-parsing test in tests/interop/http/receipts_test.go (extend: validate receipts from Go can be parsed by TypeScript and vice versa)
- [ ] T058 [P] [US6] Implement receipt proof validation test in tests/interop/http/receipts_test.go (extend: validate receipts with inclusion proofs from either implementation are successfully verified against checkpoints by both implementations)
- [ ] T059 [P] [US6] Implement receipt CBOR encoding test in tests/interop/http/receipts_test.go (extend: validate CBOR encoding from either implementation is correctly decoded by the other)
- [ ] T060 [US6] Implement database schema compatibility test in tests/interop/storage/database_test.go (FR-029, FR-030: validate both implementations use compatible SQLite schemas with matching table names, column types in snake_case, and equivalent stored metadata)
- [ ] T061 [P] [US6] Implement tile storage compatibility test in tests/interop/storage/tiles_test.go (FR-031: validate tile storage layouts follow identical C2SP naming conventions)
- [ ] T062 [P] [US6] Implement checkpoint history compatibility test in tests/interop/storage/checkpoint_test.go (FR-032: validate checkpoint history contains equivalent signed tree head data)

**Checkpoint**: At this point, all priority user stories tested - receipt and storage compatibility validated

---

## Phase 9: End-to-End Workflow Testing

**Purpose**: Validate complete workflows combining CLI and server operations across implementations

**Functional Requirements Covered**: FR-033 through FR-037

### Test Implementation for E2E Workflows

- [ ] T063 [P] [E2E] Implement pure Go workflow test in tests/interop/e2e/pure_go_test.go (FR-033: key generation → statement signing → registration → receipt retrieval → proof verification using Go CLI + Go Server, validate all operations succeed)
- [ ] T064 [P] [E2E] Implement pure TypeScript workflow test in tests/interop/e2e/pure_ts_test.go (FR-034: complete workflow using TypeScript CLI + TypeScript Server, validate all operations succeed)
- [ ] T065 [P] [E2E] Implement Go CLI + TypeScript Server workflow test in tests/interop/e2e/cross_go_ts_test.go (FR-035: cross-implementation workflow, validate all operations succeed)
- [ ] T066 [P] [E2E] Implement TypeScript CLI + Go Server workflow test in tests/interop/e2e/cross_ts_go_test.go (FR-036: cross-implementation workflow, validate all operations succeed)
- [ ] T067 [E2E] Implement workflow state equivalence test in tests/interop/e2e/state_test.go (FR-037: validate all workflows produce equivalent final states with matching tree_size, statement_count, receipt availability)

**Checkpoint**: All workflows tested - complete end-to-end cross-implementation validation

---

## Phase 10: Test Suite Validation and Reporting

**Purpose**: Validate test suite execution meets success criteria and generates comprehensive reports

**Success Criteria Covered**: SC-001 through SC-012

### Test Suite Validation

- [ ] T068 [P] Validate test execution requirements in tests/interop/lib/validate_test.go (FR-001, FR-002, FR-003: verify test suite runs in CI and locally, uses isolated environments, cleans test data)
- [ ] T069 [P] Validate RFC compliance reporting in tests/interop/lib/validate_test.go (extend: FR-004, FR-005: verify test suite validates against RFCs and reports violations with section references)
- [ ] T070 Measure test suite performance in tests/interop/performance_test.go (SC-007: validate complete test suite executes in <5 minutes, SC-012: validate throughput consistency between implementations)

### Test Report Generation

- [ ] T071 Generate comprehensive test report in tests/interop/lib/report.go (extend: implement full report generation with pass/fail counts, failed test details with diffs, RFC compliance violations, performance metrics following test-report.schema.json)
- [ ] T072 [P] Create test report templates in tests/interop/templates/ (Markdown template for stakeholder-friendly reports, CI integration format)
- [ ] T073 [P] Implement CI integration in .github/workflows/integration-tests.yml (run test suite on PR, upload test results as artifacts, fail on incompatibilities)

**Checkpoint**: Test suite validated and ready for continuous integration

---

## Phase 11: Polish & Cross-Cutting Concerns

**Purpose**: Improvements and documentation that enhance test suite usability

- [ ] T074 [P] Enhance error reporting with colored output and detailed diffs in tests/interop/lib/output.go
- [ ] T075 [P] Add test execution progress indicators in tests/interop/lib/output.go (extend: show which test category is running, progress percentage)
- [ ] T076 [P] Create test suite documentation in tests/interop/ARCHITECTURE.md (document test orchestration strategy, fixture generation process, comparison methodology)
- [ ] T077 [P] Add developer troubleshooting guide in tests/interop/TROUBLESHOOTING.md (common test failures, debugging techniques, environment setup issues)
- [ ] T078 Validate quickstart.md against actual test execution (manual: follow quickstart guide, verify all commands work as documented)
- [ ] T079 Run full integration test suite validation (manual: execute complete test suite, verify all FRs covered, all SCs met)
- [ ] T080 Create test suite demo video or asciicast (optional: demonstrate running tests locally and interpreting results)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user story tests
- **User Story 1 (Phase 3)**: Depends on Foundational phase completion
- **User Story 2 (Phase 4)**: Depends on Foundational phase completion (can run parallel with US1)
- **User Story 3 (Phase 5)**: Depends on Foundational phase completion (can run parallel with US1, US2)
- **User Story 4 (Phase 6)**: Depends on Foundational phase completion (can run parallel with US1-US3)
- **User Story 5 (Phase 7)**: Depends on Foundational + US1 (extends query tests)
- **User Story 6 (Phase 8)**: Depends on Foundational + US1 (extends receipt tests)
- **E2E Workflows (Phase 9)**: Depends on Foundational completion (can run parallel with user story phases)
- **Validation (Phase 10)**: Depends on all test phases being complete
- **Polish (Phase 11)**: Depends on all desired tests being implemented

### User Story Dependencies

- **US1 (P1)**: Can start after Foundational - No dependencies on other stories
- **US2 (P1)**: Can start after Foundational - Can run parallel with US1
- **US3 (P2)**: Can start after Foundational - Can run parallel with US1, US2
- **US4 (P2)**: Can start after Foundational - Can run parallel with US1-US3
- **US5 (P3)**: Depends on US1 for query endpoint tests - Extends existing tests
- **US6 (P3)**: Depends on US1 for receipt tests - Extends existing tests

### Parallel Opportunities

**Phase 1: Setup**
- Tasks T002-T006: All [P] (different directories/files)

**Phase 2: Foundational**
- Port allocation, CLI invocation (T008-T009): Both [P]
- Fixture generation tools (T012, T015-T016): All [P]
- Comparison and validation (T018-T019): Both [P]
- Shell scripts (T021-T022, T024): All [P]

**Phase 3: User Story 1**
- All HTTP endpoint tests (T025-T030): All [P] (different endpoints)
- Error tests (T031): Can run parallel with other US1 tests

**Phase 4: User Story 2**
- All CLI tests (T032-T037): All [P] (different commands)

**Phase 5: User Story 3**
- All crypto tests (T039-T042): All [P] (different test files)

**Phase 6: User Story 4**
- Inclusion and consistency tests (T044-T047, T049): All [P]

**Phase 7: User Story 5**
- All query tests (T051-T054): All [P]

**Phase 8: User Story 6**
- Receipt tests (T056-T059): All [P]
- Storage tests (T061-T062): Both [P]

**Phase 9: E2E**
- All workflow tests (T063-T066): All [P]

**Phase 10: Validation**
- Validation and reporting (T068-T069, T072-T073): All [P]

**Phase 11: Polish**
- All documentation (T074-T077): All [P]

---

## Parallel Example: User Story 1 (HTTP API Testing)

```bash
# After Foundational phase completes, run all US1 tests in parallel:
cd tests/interop

# Run HTTP API tests concurrently (different endpoints = no conflicts)
go test -v -parallel 6 ./http/config_test.go &
go test -v -parallel 6 ./http/entries_test.go &
go test -v -parallel 6 ./http/checkpoint_test.go &
go test -v -parallel 6 ./http/health_test.go &
go test -v -parallel 6 ./http/query_test.go &
go test -v -parallel 6 ./http/errors_test.go &
wait

# All US1 tests complete in parallel - API parity validated
```

---

## Implementation Strategy

### MVP First (User Stories 1 & 2 Only)

1. Complete Phase 1: Setup (6 tasks)
2. Complete Phase 2: Foundational (18 tasks - CRITICAL)
3. Complete Phase 3: User Story 1 (7 tasks)
4. Complete Phase 4: User Story 2 (7 tasks)
5. **STOP and VALIDATE**: Test HTTP API and CLI parity independently
6. Deliver MVP integration test suite covering most critical compatibility

### Incremental Delivery

1. Foundation (Setup + Foundational) → Test infrastructure ready
2. Add User Story 1 → Test independently → **HTTP API parity validated!**
3. Add User Story 2 → Test independently → **CLI parity validated!**
4. Add User Story 3 → Test independently → **Crypto interoperability validated!**
5. Add User Story 4 → Test independently → **Merkle proof compatibility validated!**
6. Add User Stories 5-6 → Test independently → **Query and receipt parity validated!**
7. Add E2E workflows → Test independently → **Complete workflows validated!**
8. Each phase adds validation coverage without breaking previous tests

### Parallel Team Strategy

With multiple developers after Foundational phase:
- **Developer A**: User Story 1 (HTTP API tests) - T025-T031
- **Developer B**: User Story 2 (CLI tests) - T032-T038
- **Developer C**: User Story 3 (Crypto tests) - T039-T043
- **Developer D**: User Story 4 (Merkle tests) - T044-T049

All user story phases can proceed in parallel after Foundational phase completes.

---

## Test Execution Requirements

### Running Tests Locally

```bash
# Prerequisites: Go 1.24+, Bun latest, both implementations built
cd tests/interop

# Run all tests
go test -v ./...

# Run with parallelism (faster, uses unique ports per test)
go test -v -parallel 10 ./...

# Run specific user story tests
go test -v ./http/...  # User Story 1
go test -v ./cli/...   # User Story 2
go test -v ./crypto/... # User Story 3

# Generate JSON report for CI
go test -json ./... > test-results.json

# Generate Markdown report
go run ./lib/report.go --input test-results.json --output report.md
```

### CI Integration

```bash
# GitHub Actions workflow
- name: Run Integration Tests
  run: |
    cd tests/interop
    go test -v -parallel 10 -json ./... > results.json
    go run ./lib/report.go --input results.json --output report.md

- name: Upload Test Results
  uses: actions/upload-artifact@v3
  with:
    name: integration-test-results
    path: tests/interop/report.md
```

---

## Notes

- **[P] tasks**: Different files, no shared state - safe to run in parallel
- **[Story] labels**: Map tasks to user stories for traceability
- **Test orchestration**: Go testing framework provides structure, parallel execution, CI integration
- **Each user story is independently testable**: Enables incremental delivery
- **Foundational phase is critical**: Must complete before any user story tests
- **Clean slate per test**: t.TempDir() ensures deterministic results
- **RFC compliance**: Test vectors from Go tlog ensure validation against specs, not implementations
- **snake_case throughout**: All JSON fields, test data, reports use snake_case per user requirement
- **hex encoding throughout**: All identifiers use lowercase hex per user requirement
- **File paths are exact**: Ready for implementation without additional context

---

## Task Summary

**Total Tasks**: 80
- **Setup**: 6 tasks
- **Foundational**: 18 tasks (BLOCKING) - test infrastructure, fixtures, RFC vectors, comparison utilities
- **User Story 1 (P1)**: 7 tasks - HTTP API compatibility
- **User Story 2 (P1)**: 7 tasks - CLI parity
- **User Story 3 (P2)**: 5 tasks - Cryptographic interoperability
- **User Story 4 (P2)**: 6 tasks - Merkle proof compatibility
- **User Story 5 (P3)**: 6 tasks - Query compatibility
- **User Story 6 (P3)**: 7 tasks - Receipt and storage compatibility
- **E2E Workflows**: 5 tasks - Complete workflow validation
- **Validation**: 3 tasks - Test suite validation
- **Polish**: 7 tasks - Documentation and enhancements

**Parallel Opportunities**: 56 tasks marked [P] (70% of tasks can run concurrently)

**Independent Test Points**: 6 (one per user story) + E2E workflows

**Suggested MVP Scope**: Phase 1 + Phase 2 + Phase 3 + Phase 4 (User Stories 1 & 2) = 38 tasks

**Functional Requirements Coverage**: All 41 FRs mapped to tasks

**Success Criteria Coverage**: All 12 SCs validated by test execution and reporting
