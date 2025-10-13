#!/bin/bash
# Setup test environment for cross-implementation testing
# Creates temporary directories and sets environment variables

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Create isolated test directories
setup_test_dirs() {
    local base_dir="${1:-/tmp/scitt-interop-$$}"

    log_info "Creating test environment at: $base_dir"

    # Create directories
    mkdir -p "$base_dir/go-impl"
    mkdir -p "$base_dir/ts-impl"
    mkdir -p "$base_dir/fixtures"
    mkdir -p "$base_dir/reports"

    # Export paths
    export SCITT_TEST_BASE_DIR="$base_dir"
    export SCITT_GO_WORK_DIR="$base_dir/go-impl"
    export SCITT_TS_WORK_DIR="$base_dir/ts-impl"
    export SCITT_FIXTURES_DIR="$base_dir/fixtures"
    export SCITT_REPORTS_DIR="$base_dir/reports"

    log_info "Test directories created:"
    log_info "  Base: $SCITT_TEST_BASE_DIR"
    log_info "  Go:   $SCITT_GO_WORK_DIR"
    log_info "  TS:   $SCITT_TS_WORK_DIR"

    echo "$base_dir"
}

# Setup environment variables for CLI paths
setup_cli_paths() {
    # Find Go CLI binary
    if [ -n "${SCITT_GO_CLI:-}" ]; then
        log_info "Using Go CLI from SCITT_GO_CLI: $SCITT_GO_CLI"
    elif [ -f "../../scitt-golang/scitt" ]; then
        export SCITT_GO_CLI="$(cd ../../scitt-golang && pwd)/scitt"
        log_info "Found Go CLI: $SCITT_GO_CLI"
    elif [ -f "../../../cmd/scitt/scitt" ]; then
        export SCITT_GO_CLI="$(cd ../../../cmd/scitt && pwd)/scitt"
        log_info "Found Go CLI: $SCITT_GO_CLI"
    else
        log_warn "Go CLI not found, tests requiring Go will fail"
        export SCITT_GO_CLI=""
    fi

    # Find TypeScript CLI
    if [ -n "${SCITT_TS_CLI:-}" ]; then
        log_info "Using TypeScript CLI from SCITT_TS_CLI: $SCITT_TS_CLI"
    elif [ -f "../../scitt-typescript/src/cli/index.ts" ]; then
        export SCITT_TS_CLI="bun run $(cd ../../scitt-typescript/src/cli && pwd)/index.ts"
        log_info "Found TypeScript CLI: $SCITT_TS_CLI"
    else
        log_warn "TypeScript CLI not found, tests requiring TypeScript will fail"
        export SCITT_TS_CLI=""
    fi
}

# Verify fixtures are available
setup_fixtures() {
    local fixtures_src="$(cd "$(dirname "$0")/.." && pwd)/fixtures"

    if [ ! -d "$fixtures_src" ]; then
        log_error "Fixtures directory not found at: $fixtures_src"
        return 1
    fi

    log_info "Copying test fixtures from: $fixtures_src"
    cp -r "$fixtures_src/"* "$SCITT_FIXTURES_DIR/"

    log_info "Fixtures copied: $(find "$SCITT_FIXTURES_DIR" -type f | wc -l) files"
}

# Allocate unique port for test server
allocate_port() {
    local start_port="${1:-20000}"
    local end_port="${2:-30000}"

    for port in $(seq "$start_port" "$end_port"); do
        if ! nc -z localhost "$port" 2>/dev/null; then
            echo "$port"
            return 0
        fi
    done

    log_error "No available ports in range $start_port-$end_port"
    return 1
}

# Cleanup test environment
cleanup_env() {
    local base_dir="${1:-$SCITT_TEST_BASE_DIR}"

    if [ -n "$base_dir" ] && [ -d "$base_dir" ]; then
        log_info "Cleaning up test environment: $base_dir"
        rm -rf "$base_dir"
    fi
}

# Main setup function
main() {
    local base_dir="${1:-}"

    # Setup directories
    base_dir=$(setup_test_dirs "$base_dir")

    # Setup CLI paths
    setup_cli_paths

    # Copy fixtures
    setup_fixtures

    log_info "Environment setup complete"
    log_info "To use this environment, run: source <(bash $0 $base_dir)"
    log_info "Or manually export the following variables:"
    log_info "  export SCITT_TEST_BASE_DIR=$SCITT_TEST_BASE_DIR"
    log_info "  export SCITT_GO_WORK_DIR=$SCITT_GO_WORK_DIR"
    log_info "  export SCITT_TS_WORK_DIR=$SCITT_TS_WORK_DIR"
    log_info "  export SCITT_FIXTURES_DIR=$SCITT_FIXTURES_DIR"
    log_info "  export SCITT_REPORTS_DIR=$SCITT_REPORTS_DIR"
    log_info "  export SCITT_GO_CLI=$SCITT_GO_CLI"
    log_info "  export SCITT_TS_CLI=$SCITT_TS_CLI"
}

# Run if executed directly
if [ "${BASH_SOURCE[0]}" == "${0}" ]; then
    main "$@"
fi
