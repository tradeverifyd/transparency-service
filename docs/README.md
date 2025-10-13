# SCITT Transparency Service API Documentation

This directory contains the OpenAPI documentation for the SCITT Transparency Service.

## Viewing the Documentation

### GitHub Pages (Recommended)

The API documentation is automatically published via GitHub Pages at:
```
https://tradeverifyd.github.io/transparency-service/
```

The documentation is rendered using Swagger UI and provides an interactive interface to explore the API.

### Local Preview

To preview the documentation locally:

1. Start a local web server in the docs directory:
   ```bash
   cd docs
   python3 -m http.server 8000
   ```

2. Open your browser to: http://localhost:8000

### Files

- `index.html` - Swagger UI interface for rendering the OpenAPI specification
- `openapi.yaml` - OpenAPI 3.0 specification for the SCITT Transparency Service API
- `.nojekyll` - Tells GitHub Pages not to process files with Jekyll

## Updating the Documentation

The OpenAPI specification is maintained in the Go implementation at:
```
scitt-golang/internal/server/openapi.yaml
```

To update the documentation:

1. Edit the OpenAPI spec in `scitt-golang/internal/server/openapi.yaml`
2. Copy the updated spec to the docs directory:
   ```bash
   cp scitt-golang/internal/server/openapi.yaml docs/openapi.yaml
   ```
3. Commit and push the changes - GitHub Pages will automatically update

## API Endpoints

The SCITT Transparency Service provides the following endpoints:

### System
- `GET /` - API Documentation (Swagger UI)
- `GET /health` - Health check
- `GET /openapi.json` - OpenAPI specification

### SCITT Configuration
- `GET /.well-known/scitt-configuration` - Service configuration
- `GET /.well-known/scitt-keys` - Service verification keys (COSE Key Set)

### Statements
- `POST /entries` - Register a new COSE Sign1 statement
- `GET /entries/{entry_id}` - Retrieve a receipt for a registered statement

## SCRAPI Compliance

This service implements the SCRAPI (SCITT Reference APIs) specification, providing:
- COSE Sign1 statement registration
- Transparent Merkle tree inclusion proofs
- RFC 6962 signed tree head checkpoints
- Well-known service discovery endpoints
