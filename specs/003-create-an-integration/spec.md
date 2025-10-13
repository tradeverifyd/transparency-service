# Feature Specification: Cross-Implementation Integration Test Suite

**Feature Branch**: `003-create-an-integration`
**Created**: 2025-10-12
**Status**: Draft
**Input**: User description: "create an integration test suite that cross tests the scitt-golang and scitt-typescript implementations, covering all critical functionality, including key generation, statement signing with hash envelope, statement registration via scrapi, receipt retrieval, tile retrieval, statement querying, inclusion and consistency proof checks. Ensure that both cli's generate keys, sign statements, register statements, retrieve receipts, verify inclusion proofs and verify consistency proofs with the same user experience. Ensure that both servers expose the same http interfaces for registering signed statements, retrieving configurations, retrieving verification keys, retrieving signed statemnets, retrieving statements, retrieving receipts, querying the log for signed statements about about a specific sub or from a specific iss, or for a specific media type."

## Clarifications

### Session 2025-10-12

- Q: Where should the cross-implementation test suite execute? → A: Both CI and local developers - provides flexibility and fast iteration
- Q: When implementations produce different but valid results, which serves as the reference? → A: External specification/RFC is reference - neither implementation is canonical
- Q: How should test databases and storage be managed between test runs? → A: Clean slate each run - delete all test data before tests start

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Cross-Implementation API Compatibility Testing (Priority: P1)

As a transparency service developer, I need to verify that both Go and TypeScript implementations expose identical HTTP APIs, ensuring that clients can seamlessly interact with either implementation without modification.

**Why this priority**: This is the foundation of dual-implementation value - API compatibility ensures users can choose implementations based on operational needs without client-side changes. Without this, the implementations diverge and become separate products.

**Independent Test**: Can be fully tested by starting both servers independently, making identical HTTP requests to each (POST statement, GET receipt, GET checkpoint, GET configuration), and comparing response formats, status codes, and semantic content. Delivers immediate value by proving API parity.

**Acceptance Scenarios**:

1. **Given** both Go and TypeScript servers are running, **When** identical transparency configuration requests are made to both servers, **Then** both return identical configuration structures with matching supported algorithms, endpoints, and origins
2. **Given** both implementations are initialized with the same origin, **When** a COSE Sign1 statement is registered to both servers, **Then** both return 201 Created with identical response schema (entryId, statementHash)
3. **Given** statements are registered in both implementations, **When** receipts are retrieved by entry ID from both servers, **Then** both return identical receipt structures with matching metadata
4. **Given** multiple statements in both transparency logs, **When** checkpoints are requested from both servers, **Then** both return signed tree heads with identical format (origin, tree size, root hash, timestamp, signature lines)
5. **Given** both servers are running, **When** health check requests are made, **Then** both return 200 OK with identical health status structures

---

### User Story 2 - CLI Tool User Experience Parity (Priority: P1)

As a transparency service user, I need both CLI tools (Go and TypeScript) to provide identical command structures, arguments, and output formats so I can switch between implementations without relearning commands or updating scripts.

**Why this priority**: CLI parity ensures operational consistency - scripts, documentation, and user workflows remain valid across implementations. This is critical for adoption and prevents fragmentation of the user base.

**Independent Test**: Can be fully tested by running equivalent CLI commands for both tools (init, serve, statement sign/verify/hash, receipt operations) and comparing command-line arguments, output formats, exit codes, and side effects (files created, database state). Delivers value by ensuring seamless user experience.

**Acceptance Scenarios**:

1. **Given** both CLI tools are installed, **When** initialization commands are run with identical arguments (origin, paths), **Then** both create equivalent directory structures, configuration files, and cryptographic key pairs
2. **Given** both CLIs have initialized services, **When** statement signing commands are executed with identical payloads and keys, **Then** both produce byte-identical COSE Sign1 structures (or semantically equivalent with different timestamps)
3. **Given** signed statements from either implementation, **When** verification commands are run with the corresponding public keys, **Then** both CLIs report verification success/failure identically
4. **Given** both CLIs have statement files, **When** hash commands are executed, **Then** both output identical statement hash formats and values
5. **Given** both CLIs can access running servers, **When** statement registration commands are executed, **Then** both successfully register and output identical response formats (entry ID, hash)

---

### User Story 3 - Cryptographic Interoperability Testing (Priority: P2)

As a security auditor, I need to verify that statements signed by one implementation can be verified by the other, ensuring cryptographic operations are compatible and compliant with COSE/RFC standards.

**Why this priority**: Cryptographic interoperability is essential for trust - users must be confident that signatures created by either implementation are valid across both. This validates RFC compliance and prevents vendor lock-in.

**Independent Test**: Can be fully tested by generating keys in one implementation, signing statements with those keys, then verifying the signatures in the other implementation. Delivers value by proving cryptographic compatibility and standards compliance.

**Acceptance Scenarios**:

1. **Given** keys generated by the Go implementation, **When** statements are signed using Go CLI and verified using TypeScript CLI, **Then** verification succeeds and signature is confirmed valid
2. **Given** keys generated by the TypeScript implementation, **When** statements are signed using TypeScript CLI and verified using Go CLI, **Then** verification succeeds and signature is confirmed valid
3. **Given** hash envelope statements created by Go, **When** verified by TypeScript with the original payload, **Then** both signature and payload hash are validated successfully
4. **Given** hash envelope statements created by TypeScript, **When** verified by Go with the original payload, **Then** both signature and payload hash are validated successfully
5. **Given** keys in JWK format from either implementation, **When** imported by the other implementation, **Then** key thumbprints match and keys can be used for sign/verify operations

---

### User Story 4 - Merkle Tree Proof Interoperability (Priority: P2)

As a transparency log auditor, I need to verify that Merkle inclusion and consistency proofs generated by one implementation can be verified by the other, ensuring cryptographic commitment schemes are compatible.

**Why this priority**: Proof interoperability enables cross-implementation auditing - auditors can use tools from either implementation to verify proofs from any server. This strengthens the transparency guarantee through implementation diversity.

**Independent Test**: Can be fully tested by registering statements in one implementation's server, retrieving inclusion/consistency proofs, then verifying those proofs using the other implementation's verification functions. Delivers value by enabling heterogeneous auditing ecosystems.

**Acceptance Scenarios**:

1. **Given** statements registered in Go server with known tree size, **When** inclusion proofs are retrieved and verified using TypeScript proof verification, **Then** proofs validate successfully and confirm statement presence
2. **Given** statements registered in TypeScript server with known tree size, **When** inclusion proofs are retrieved and verified using Go proof verification, **Then** proofs validate successfully and confirm statement presence
3. **Given** Go server has grown from size N to M, **When** consistency proof from N to M is retrieved and verified using TypeScript, **Then** proof validates and confirms tree evolution consistency
4. **Given** TypeScript server has grown from size N to M, **When** consistency proof from N to M is retrieved and verified using Go, **Then** proof validates and confirms tree evolution consistency
5. **Given** tree roots computed by Go server, **When** tree roots for same tree size are computed independently by TypeScript, **Then** root hashes match exactly

---

### User Story 5 - Statement Query Compatibility (Priority: P3)

As a supply chain verifier, I need to query transparency logs using issuer, subject, or media type filters, and receive consistent results regardless of which implementation hosts the log.

**Why this priority**: Query compatibility enables consistent discovery across implementations - users can search for statements using the same criteria regardless of server implementation. This is important for usability but lower priority than core operations.

**Independent Test**: Can be fully tested by populating both implementations with identical statement sets (varying issuers, subjects, media types), then executing equivalent queries and comparing result sets for consistency. Delivers value by enabling predictable statement discovery.

**Acceptance Scenarios**:

1. **Given** both servers contain statements from multiple issuers, **When** queries filter by specific issuer, **Then** both implementations return the same statements matching that issuer
2. **Given** both servers contain statements about multiple subjects, **When** queries filter by specific subject, **Then** both implementations return the same statements matching that subject
3. **Given** both servers contain statements with different media types, **When** queries filter by specific content type, **Then** both implementations return the same statements matching that media type
4. **Given** both servers contain 100+ statements, **When** paginated queries are executed with identical parameters, **Then** both implementations return consistent result ordering and pagination behavior
5. **Given** query filters combining multiple criteria (issuer AND subject), **When** applied to both servers, **Then** both return identical filtered result sets

---

### User Story 6 - Receipt Format Compatibility (Priority: P3)

As a verifier, I need receipts from both implementations to contain equivalent information in compatible formats, enabling receipt verification regardless of which implementation generated it.

**Why this priority**: Receipt format compatibility enables portable verification - a receipt from any server can be verified by any client. This is important for ecosystem interoperability but lower priority than core transparency operations.

**Independent Test**: Can be fully tested by registering identical statements to both implementations, retrieving receipts, then comparing receipt structures for semantic equivalence (entry ID, statement hash, tree size, inclusion proof format). Delivers value by enabling universal receipt verification.

**Acceptance Scenarios**:

1. **Given** identical statements registered in both implementations, **When** receipts are retrieved for those statements, **Then** receipts contain equivalent metadata (entry ID, statement hash, tree size, timestamp)
2. **Given** receipts from Go server, **When** parsed by TypeScript receipt verification functions, **Then** receipts are successfully decoded and validated
3. **Given** receipts from TypeScript server, **When** parsed by Go receipt verification functions, **Then** receipts are successfully decoded and validated
4. **Given** receipts with inclusion proofs from either implementation, **When** verified against the corresponding checkpoint, **Then** both implementations confirm proof validity
5. **Given** receipt CBOR encoding from either implementation, **When** decoded by the other, **Then** all fields are correctly interpreted and match expected values

---

### Edge Cases

- **Empty Tree State**: What happens when querying checkpoints, receipts, or proofs from an empty (newly initialized) transparency log in each implementation?
- **Concurrent Registration**: How do both implementations handle simultaneous statement registration requests? Are tree sizes and entry IDs consistent?
- **Large Payloads**: How do both implementations handle COSE Sign1 statements with maximum payload sizes or hash envelopes with very large referenced artifacts?
- **Key Format Variations**: What happens when JWK keys have optional fields populated differently, or PEM keys have different encoding styles?
- **Invalid Proofs**: How consistently do both implementations reject tampered inclusion/consistency proofs?
- **Tree Size Mismatches**: What happens when requesting inclusion proofs for tree sizes that don't match the current tree state?
- **Network Failures**: How do CLI tools in both implementations handle server unavailability during statement registration or receipt retrieval?
- **Character Encoding**: How do both implementations handle non-ASCII characters in issuer URLs, subjects, or payload content?
- **Timestamp Variations**: How do implementations handle statements with timestamps in the future or distant past?
- **Key Reuse**: What happens when the same keypair is used across both implementations to initialize separate services?

## Requirements *(mandatory)*

### Functional Requirements

#### Test Execution Requirements

- **FR-001**: Test suite MUST be executable both in CI/CD pipelines and on local developer machines, ensuring consistent validation across environments
- **FR-002**: Test suite MUST provide isolated test environments with temporary directories and ports to enable parallel test execution without conflicts
- **FR-003**: Test suite MUST clean all test data (databases, storage directories, configuration files) before each test run to ensure deterministic, repeatable results
- **FR-004**: Test suite MUST validate both implementations against external specifications (RFCs 9052, 6962, 8392, 9597, 7638, C2SP tlog-tiles) as the authoritative reference, not against each other
- **FR-005**: Test suite MUST report specification violations with references to the specific RFC section or specification clause that was violated

#### CLI Compatibility Requirements

- **FR-006**: Test suite MUST validate that both Go and TypeScript CLIs accept identical command-line arguments for all operations (init, serve, statement sign/verify/hash, receipt operations)
- **FR-007**: Test suite MUST verify that both CLIs produce equivalent output formats (JSON structure, error messages, success confirmations) for each command
- **FR-008**: Test suite MUST confirm that both CLIs create compatible configuration files (YAML format, field names, default values)
- **FR-009**: Test suite MUST validate that both CLIs generate interoperable cryptographic key pairs (PEM private keys, JWK public keys with RFC 7638 thumbprints)
- **FR-010**: Test suite MUST verify that statement signing operations in both CLIs produce COSE Sign1 structures that are mutually verifiable

#### HTTP API Compatibility Requirements

- **FR-011**: Test suite MUST validate that `POST /entries` endpoints in both implementations accept identical request formats (Content-Type: application/cose) and return identical response schemas
- **FR-012**: Test suite MUST verify that `GET /entries/{id}` endpoints in both implementations return receipts with equivalent structures and semantic content
- **FR-013**: Test suite MUST confirm that `GET /checkpoint` endpoints in both implementations return checkpoints with identical signed note format
- **FR-014**: Test suite MUST validate that `GET /.well-known/transparency-configuration` endpoints in both implementations return equivalent configuration structures
- **FR-015**: Test suite MUST verify that `GET /health` endpoints in both implementations return compatible health status information
- **FR-016**: Test suite MUST validate that statement query endpoints (filtering by issuer, subject, content type) in both implementations return equivalent result sets for identical queries

#### Cryptographic Compatibility Requirements

- **FR-017**: Test suite MUST verify that COSE Sign1 statements signed by Go implementation can be successfully verified by TypeScript implementation using the same public key
- **FR-018**: Test suite MUST verify that COSE Sign1 statements signed by TypeScript implementation can be successfully verified by Go implementation using the same public key
- **FR-019**: Test suite MUST validate that hash envelope statements created by either implementation contain equivalent structures (hash algorithm, hash value, location, content type)
- **FR-020**: Test suite MUST confirm that hash envelope verification in both implementations accepts statements from either implementation and validates payload hashes correctly
- **FR-021**: Test suite MUST verify that JWK public keys exported by either implementation can be imported and used by the other implementation
- **FR-022**: Test suite MUST validate that JWK thumbprints (RFC 7638) computed by both implementations for the same key are byte-identical

#### Merkle Tree Compatibility Requirements

- **FR-023**: Test suite MUST verify that inclusion proofs generated by Go server can be successfully verified by TypeScript verification functions
- **FR-024**: Test suite MUST verify that inclusion proofs generated by TypeScript server can be successfully verified by Go verification functions
- **FR-025**: Test suite MUST validate that consistency proofs for tree growth in Go server can be verified by TypeScript
- **FR-026**: Test suite MUST validate that consistency proofs for tree growth in TypeScript server can be verified by Go
- **FR-027**: Test suite MUST confirm that tree root hashes computed by both implementations for identical leaf sets are byte-identical
- **FR-028**: Test suite MUST verify that tile naming conventions (C2SP tlog-tiles) are identical across implementations for the same entry IDs

#### Database and Storage Compatibility Requirements

- **FR-029**: Test suite MUST validate that both implementations use compatible SQLite schema structures (table names, column types, indexes)
- **FR-030**: Test suite MUST verify that statement metadata stored by both implementations contains equivalent fields and values
- **FR-031**: Test suite MUST confirm that tile storage layouts in both implementations follow identical C2SP naming conventions
- **FR-032**: Test suite MUST validate that checkpoint history stored by both implementations contains equivalent signed tree head data

#### End-to-End Workflow Requirements

- **FR-033**: Test suite MUST execute complete workflows (key generation → statement signing → registration → receipt retrieval → proof verification) using Go CLI + Go Server and verify success
- **FR-034**: Test suite MUST execute complete workflows using TypeScript CLI + TypeScript Server and verify success
- **FR-035**: Test suite MUST execute cross-implementation workflows (Go CLI + TypeScript Server) and verify success
- **FR-036**: Test suite MUST execute cross-implementation workflows (TypeScript CLI + Go Server) and verify success
- **FR-037**: Test suite MUST validate that workflows produce equivalent final states (tree sizes, statement counts, receipt availability) across implementations

#### Error Handling Requirements

- **FR-038**: Test suite MUST verify that both implementations return equivalent HTTP status codes for error conditions (400 Bad Request for invalid COSE, 404 Not Found for missing entries, 500 Internal Server Error for system failures)
- **FR-039**: Test suite MUST validate that error response bodies in both implementations contain equivalent error information (error codes, messages, details)
- **FR-040**: Test suite MUST confirm that both CLIs exit with equivalent error codes for failure scenarios
- **FR-041**: Test suite MUST verify that both implementations handle malformed inputs consistently (invalid CBOR, corrupted signatures, missing headers)

### Key Entities

- **Test Configuration**: Represents shared configuration for running cross-implementation tests, including server ports, temporary directories, timeout values, and test data sets. Attributes: go_server_port, ts_server_port, test_data_path, timeout_seconds.

- **Implementation Under Test**: Represents one of the two implementations (Go or TypeScript) with metadata about its location, startup commands, and capabilities. Attributes: name (go/typescript), binary_path, source_path, cli_command, server_command.

- **Test Statement**: Represents a COSE Sign1 statement used in testing, including payload content, signing key, CWT claims, and expected hash values. Attributes: payload_bytes, issuer, subject, content_type, signature_bytes, statement_hash.

- **Test Keypair**: Represents a cryptographic keypair used across implementations, including private key (PEM format), public key (JWK format), and thumbprint. Attributes: private_key_pem, public_key_jwk, jwk_thumbprint, algorithm (ES256).

- **Cross-Implementation Test Result**: Represents the outcome of testing equivalent operations across both implementations. Attributes: test_name, go_result, ts_result, equivalence_check_passed, differences_detected.

- **HTTP API Test Case**: Represents a specific HTTP request/response test executed against both servers. Attributes: endpoint, http_method, request_body, expected_status_code, response_schema, go_response, ts_response.

- **Proof Test Vector**: Represents a Merkle proof (inclusion or consistency) with associated test data for cross-verification. Attributes: proof_type (inclusion/consistency), tree_size, leaf_index, proof_hashes, root_hash, source_implementation, verification_result_per_implementation.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Both implementations successfully complete initialization operations within 2 seconds and produce compatible directory structures with identical file types
- **SC-002**: 100% of equivalent CLI commands (minimum 20 command variations tested) produce semantically equivalent outputs across implementations
- **SC-003**: 100% of HTTP API endpoints (minimum 10 endpoint variations tested) return compatible responses with matching HTTP status codes and equivalent JSON/CBOR structures
- **SC-004**: Cross-implementation cryptographic verification succeeds for 100% of test cases (minimum 50 sign/verify combinations per direction)
- **SC-005**: Merkle proof cross-verification succeeds for 100% of test cases (minimum 30 proofs tested per direction for inclusion and consistency)
- **SC-006**: Complete end-to-end workflows (key generation through proof verification) execute successfully in under 30 seconds for both pure-implementation and cross-implementation scenarios
- **SC-007**: Test suite detects and reports any API incompatibilities within 5 minutes of execution, with clear diff outputs showing discrepancies
- **SC-008**: Query operations return identical result sets (100% match on statement IDs and metadata) when filtering by issuer, subject, or content type across implementations
- **SC-009**: Receipt formats from both implementations are successfully parsed and validated by the other implementation in 100% of test cases (minimum 20 receipts tested)
- **SC-010**: Test suite executes fully automated without manual intervention, producing a comprehensive compatibility report with pass/fail status for all test categories
- **SC-011**: Both implementations handle error conditions identically, returning matching HTTP status codes and equivalent error structures for 100% of error scenarios tested (minimum 15 error types)
- **SC-012**: Statement registration rate is consistent across implementations, with throughput differing by no more than 50% under identical load conditions (100 concurrent registrations)
