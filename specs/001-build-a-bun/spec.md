# Feature Specification: IETF Standards-Based Transparency Service

**Feature Branch**: `001-build-a-bun`
**Created**: 2025-10-11
**Status**: Draft
**Input**: User description: "build a bun library and cli that can implement the critical operations of a transparency service using standards from the ietf, including: https://datatracker.ietf.org/doc/draft-ietf-cose-hash-envelope/ https://datatracker.ietf.org/doc/draft-ietf-scitt-architecture/ https://datatracker.ietf.org/doc/draft-ietf-scitt-scrapi/ and base your cose implementations off of https://github.com/transmute-industries/cose, but only implement what is necessary for the transparency service to operate. The end goal is a set of simple cli commands that can be used to standup a transparency service which is backed by SQL, including all of the identity management via URLs and signatures using COSE Sign1, the system must also store all the records associated with the receipts from the merkle tree using a mino compatible interface that enables object storage portability"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Service Operator Deploys Transparency Service (Priority: P1)

A service operator needs to deploy and configure a standards-compliant transparency service to maintain an auditable record of supply chain statements.

**Why this priority**: Without a running transparency service, no other functionality is possible. This is the foundational capability that enables all other user stories.

**Independent Test**: Can be fully tested by deploying the service, verifying it starts successfully, responds to health checks, and accepts configuration for SQL database and object storage connections.

**Acceptance Scenarios**:

1. **Given** the CLI tools are installed, **When** operator runs service initialization command with database and storage configuration, **Then** the service starts and creates necessary database tables
2. **Given** a running service, **When** operator queries the transparency configuration endpoint, **Then** service returns its capabilities, supported algorithms, and registration policies
3. **Given** service deployment completes, **When** operator runs health check command, **Then** service reports healthy status for database, object storage, and core service components

---

### User Story 2 - Issuer Registers Large Artifact by Hash (Priority: P2)

An issuer (software publisher, manufacturer, or supply chain participant) needs to register large artifacts (such as parquet files, container images, or datasets) in the transparency service without uploading the entire artifact content to the service.

**Why this priority**: This is the primary write operation for the transparency service. Hash envelope approach enables efficient registration of arbitrarily large artifacts, making this the core value proposition.

**Independent Test**: Can be tested by computing a hash of a large file (e.g., 1GB parquet file), creating a signed hash envelope statement using the CLI, submitting it to the service, and verifying the service returns a receipt proving registration.

**Acceptance Scenarios**:

1. **Given** an issuer has an artifact file (e.g., parquet file of any size), **When** issuer runs CLI command to create hash envelope statement with issuer identity and artifact path, **Then** CLI computes artifact hash, creates COSE Sign1 signed hash envelope statement without including full artifact content
2. **Given** a signed hash envelope statement, **When** issuer submits the statement to the transparency service via CLI registration command, **Then** service validates the hash envelope structure and signature, returns either immediate receipt or registration tracking identifier
3. **Given** statement submission returns a tracking identifier, **When** issuer polls registration status using the CLI, **Then** service eventually returns a final receipt containing Merkle tree inclusion proof
4. **Given** a receipt is issued, **When** issuer retrieves the receipt, **Then** receipt contains proof of inclusion, tree position, timestamp, and can be combined with original statement to create transparent statement
5. **Given** multiple large artifacts (parquet files ranging from MB to GB), **When** issuer registers them all, **Then** each registration completes in consistent time regardless of artifact size since only hash is registered

---

### User Story 3 - Verifier Checks Statement Authenticity for Large Artifacts (Priority: P3)

A verifier (consumer, auditor, or relying party) receives an artifact with its transparent statement (hash envelope + receipt) and needs to verify the artifact's authenticity and inclusion in the transparency log without contacting the original issuer.

**Why this priority**: Verification is the consumer side of the transparency service. While critical for the ecosystem, it depends on having statements already registered (P2), making it lower priority for initial MVP.

**Independent Test**: Can be tested by receiving an artifact file, transparent statement (hash envelope with receipt), then using CLI verification commands to validate the artifact hash matches the hash envelope, signature is valid, and Merkle inclusion proof is correct independently.

**Acceptance Scenarios**:

1. **Given** a verifier receives an artifact file and transparent statement, **When** verifier runs CLI command to verify the artifact against the hash envelope, **Then** CLI computes artifact hash and confirms it matches the hash in the hash envelope
2. **Given** a transparent statement with receipt, **When** verifier runs CLI verification command, **Then** the COSE Sign1 signature is validated against the issuer's public key (retrieved from issuer URL) and the Merkle inclusion proof is verified against the service's commitment
3. **Given** a statement identifier, **When** verifier queries the service to resolve the receipt, **Then** service returns the complete receipt with inclusion proof from object storage
4. **Given** a transparent statement, **When** verifier inspects the metadata using CLI, **Then** issuer identity (URL), timestamp, subject, hash algorithm, and artifact content type are displayed
5. **Given** only the artifact file and transparent statement (no network access), **When** verifier runs offline verification command, **Then** CLI verifies hash match, signature validity, and receipt structure without querying the service

---

### User Story 4 - Auditor Reviews Service Log Consistency (Priority: P4)

An auditor (independent monitor or compliance officer) examines the transparency service's append-only log to verify consistency, detect unauthorized modifications, and ensure service operates correctly.

**Why this priority**: Auditing is essential for long-term trust but is not required for basic operation. It's a higher-level governance capability built on top of the core registration and verification flows.

**Independent Test**: Can be tested by registering multiple statements, then using CLI audit commands to verify the Merkle tree is consistent, append-only, and matches published commitments.

**Acceptance Scenarios**:

1. **Given** a running transparency service, **When** auditor requests service log metadata via CLI, **Then** service returns current tree size, root hash, and checkpoint information
2. **Given** two sequential service checkpoints, **When** auditor verifies consistency between checkpoints using CLI, **Then** the consistency proof demonstrates the log is append-only and no entries were removed or modified
3. **Given** access to SQL database and object storage, **When** auditor runs integrity check command, **Then** CLI verifies all receipt hashes in SQL match object storage content
4. **Given** service publishes periodic signed tree heads, **When** auditor validates the signed tree head using CLI, **Then** the service's signature over the tree commitment is cryptographically valid

---

### Edge Cases

- What happens when a statement submission includes invalid COSE Sign1 structure or unsupported hash algorithm?
- How does the system handle duplicate statement submissions (same hash envelope, same issuer)?
- What happens when object storage is temporarily unavailable during receipt storage?
- How does the service handle malformed registration requests or corrupted hash envelope payloads?
- What happens when SQL database and object storage become out of sync?
- How does verification work when an issuer's key has been rotated or revoked?
- What happens when a verifier requests a receipt for a statement that is still being processed (pending state)?
- How does the system handle hash envelope verification when the verifier receives an artifact that doesn't match the hash in the statement?
- What happens when a verifier cannot retrieve the issuer's public key from the issuer URL?
- How does the service handle extremely high registration rates that could impact Merkle tree computation performance?

## Requirements *(mandatory)*

### Functional Requirements

**Service Deployment & Configuration**

- **FR-001**: System MUST provide CLI commands to initialize and start a transparency service instance
- **FR-002**: System MUST support configuration of SQL database connection (PostgreSQL, MySQL, or SQLite)
- **FR-003**: System MUST support configuration of object storage using MinIO-compatible interface (S3 API)
- **FR-004**: System MUST create and maintain database schema for statements, receipts, and Merkle tree nodes
- **FR-005**: System MUST expose a health check endpoint reporting status of database, object storage, and service components

**Identity Management**

- **FR-006**: System MUST support issuer identities identified by URLs (e.g., https://example.com/issuer)
- **FR-007**: System MUST use COSE Sign1 signatures for all signed statements
- **FR-008**: System MUST support ES256 (ECDSA with SHA-256) as the minimum required signature algorithm
- **FR-009**: System MUST validate COSE Sign1 signatures during statement registration
- **FR-010**: System MUST retrieve issuer public keys from issuer URLs during verification

**Hash Envelope & Statement Registration**

- **FR-011**: System MUST use COSE Hash Envelope format for ALL signed statements
- **FR-012**: System MUST provide CLI command to create hash envelope statements from artifact files of any size
- **FR-013**: System MUST compute artifact hashes without loading entire file into memory (streaming hash computation)
- **FR-014**: System MUST support SHA-256 as the minimum required hash algorithm
- **FR-015**: System MUST include payload hash algorithm, content type, and optional payload location in hash envelope
- **FR-016**: System MUST provide CLI command to submit signed hash envelope statements to the transparency service
- **FR-017**: System MUST implement SCITT registration endpoint accepting COSE-formatted hash envelope statements
- **FR-018**: System MUST validate hash envelope structure, hash algorithm, and signature before accepting registration
- **FR-019**: System MUST assign each registered statement a unique identifier
- **FR-020**: System MUST add registered statements to an append-only Merkle tree (registering hash envelope, not original artifact)
- **FR-021**: System MUST generate receipts containing Merkle inclusion proofs for registered statements
- **FR-022**: System MUST store receipt data in object storage via MinIO-compatible interface
- **FR-023**: System MUST store receipt metadata (hash, location, tree position) in SQL database
- **FR-024**: System MUST support both synchronous receipt issuance and asynchronous registration with status polling

**Statement Verification**

- **FR-025**: System MUST provide CLI command to compute and verify artifact hash matches hash envelope
- **FR-026**: System MUST provide CLI command to verify COSE Sign1 signatures on hash envelope statements
- **FR-027**: System MUST provide CLI command to verify Merkle inclusion proofs in receipts
- **FR-028**: System MUST implement receipt resolution endpoint for retrieving receipts by statement identifier
- **FR-029**: System MUST provide CLI command to retrieve and display receipt details from object storage
- **FR-030**: Users MUST be able to verify transparent statements offline (without querying the service) if they have the artifact, statement, and receipt
- **FR-031**: System MUST support verification workflow: verify artifact hash → verify signature → verify receipt

**Transparency Service Operations**

- **FR-032**: System MUST expose transparency configuration endpoint describing service capabilities
- **FR-033**: System MUST maintain append-only log property (no deletion or modification of registered statements)
- **FR-034**: System MUST provide CLI command to query current tree size and root hash
- **FR-035**: System MUST generate consistency proofs between different tree states for auditing
- **FR-036**: System MUST handle registration of hash envelopes for large artifacts (GB+ size) efficiently by never requiring full artifact upload to service

**Data Portability**

- **FR-037**: System MUST store all receipt data in object storage (not just in SQL database)
- **FR-038**: System MUST ensure receipt data in object storage can be migrated between MinIO-compatible providers
- **FR-039**: System MUST use content-addressable storage (hash-based keys) for receipt objects
- **FR-040**: System MUST store hash envelope statements (not original artifacts) in the transparency log

### Key Entities

- **Issuer**: An entity (identified by URL) that creates and signs hash envelope statements about artifacts. Has a cryptographic key pair for signing and serves public key at identity URL.
- **Artifact**: The actual file or data being registered (e.g., parquet file, container image, dataset). Not stored in the transparency service, only its hash is registered.
- **Hash Envelope**: A COSE Sign1 signed structure containing the hash of an artifact instead of the full artifact, enabling efficient registration of large files. Includes hash algorithm, content type, and optional payload location.
- **Statement**: A signed hash envelope assertion about an artifact, including issuer identity, subject, content hash, hash algorithm, and metadata. Always encoded as COSE Sign1 with hash envelope structure.
- **Receipt**: Cryptographic proof that a hash envelope statement has been registered in the transparency log. Contains Merkle inclusion proof, tree size, timestamp, and statement identifier.
- **Transparent Statement**: A hash envelope statement combined with its receipt, proving the statement is registered in the transparency service. Can be verified offline with the original artifact.
- **Merkle Tree**: Append-only cryptographic data structure maintaining all registered hash envelope statements. Enables verification of inclusion and consistency.
- **Service Configuration**: Describes transparency service capabilities, supported hash algorithms, signature algorithms, and registration policies.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Service operators can deploy a running transparency service in under 10 minutes with a single CLI command
- **SC-002**: Issuers can create and register a hash envelope statement for a 1GB artifact in under 30 seconds
- **SC-003**: Issuers can create and register a hash envelope statement for a 10MB artifact in under 5 seconds
- **SC-004**: Verifiers can verify artifact hash, signature, and receipt inclusion proof in under 2 seconds regardless of original artifact size
- **SC-005**: Service maintains append-only log property with 100% consistency (no unauthorized modifications detected in audits)
- **SC-006**: Object storage and SQL database remain synchronized with 99.9% consistency (receipt hashes match stored content)
- **SC-007**: System handles registration of hash envelopes for artifacts ranging from 1KB to 100GB without requiring full artifact upload
- **SC-008**: System handles at least 100 concurrent hash envelope registrations without degradation
- **SC-009**: All receipts stored in object storage can be migrated to a different MinIO-compatible provider without data loss
- **SC-010**: CLI tools provide clear error messages with actionable guidance when operations fail (measured by user task completion without support)
- **SC-011**: Verifiers can verify transparent statements completely offline (90%+ of verification tasks completed without network access)

## Assumptions

- SQL database is already provisioned and accessible (not managing database installation)
- MinIO-compatible object storage is already deployed or accessible (not managing object storage deployment)
- Network connectivity between service, database, and object storage is reliable
- Issuers manage their own key pairs and identity URLs (service does not provide key management)
- Issuers serve their public keys at their identity URLs in a standard format
- Artifacts are stored and distributed by issuers or third-party storage (service never stores original artifacts)
- Verifiers have access to both the artifact and its transparent statement (hash envelope + receipt)
- Initial deployment supports single-node operation (distributed/replicated deployment is future work)
- Service operates in trusted environment (TLS termination and authentication handled by reverse proxy or API gateway)
- Hash envelope statements reference artifact hash and content type; original artifact location is optional metadata
- All registered statements use hash envelope format (no support for embedding full payload in COSE Sign1)

## Standards Compliance

This feature implements the following IETF draft specifications:

- **COSE Hash Envelope** (draft-ietf-cose-hash-envelope): Enables all statements to reference artifacts by hash, supporting efficient registration of large files
- **SCITT Architecture** (draft-ietf-scitt-architecture): Defines transparency service roles, operations, and data structures
- **SCITT SCRAPI** (draft-ietf-scitt-scrapi): Specifies REST API for hash envelope statement registration and receipt resolution

COSE cryptographic operations based on the transmute-industries/cose library capabilities.
