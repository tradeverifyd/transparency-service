#!/bin/bash
# Build both Go and TypeScript SCITT implementations
# Verifies that binaries are ready for testing

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Find repository root
find_repo_root() {
    local dir="$(cd "$(dirname "$0")/.." && pwd)"
    while [ "$dir" != "/" ]; do
        if [ -f "$dir/.git/config" ] || [ -d "$dir/.git" ]; then
            echo "$dir"
            return 0
        fi
        dir="$(dirname "$dir")"
    done
    echo "."
}

# Build Go implementation
build_go() {
    log_step "Building Go SCITT implementation..."

    local repo_root=$(find_repo_root)
    local go_impl_dir="$repo_root/scitt-golang"

    # Try alternative locations
    if [ ! -d "$go_impl_dir" ]; then
        go_impl_dir="$repo_root"
        if [ ! -f "$go_impl_dir/go.mod" ]; then
            log_error "Go implementation not found. Searched:"
            log_error "  - $repo_root/scitt-golang"
            log_error "  - $repo_root"
            return 1
        fi
    fi

    log_info "Found Go implementation at: $go_impl_dir"

    # Check if main.go or cmd/scitt/main.go exists
    local main_path=""
    if [ -f "$go_impl_dir/cmd/scitt/main.go" ]; then
        main_path="$go_impl_dir/cmd/scitt"
    elif [ -f "$go_impl_dir/main.go" ]; then
        main_path="$go_impl_dir"
    else
        log_error "Cannot find main.go in expected locations"
        return 1
    fi

    # Build the binary
    log_info "Building Go binary from: $main_path"
    cd "$main_path"

    if go build -o scitt .; then
        log_info "Go build successful: $(pwd)/scitt"
        export SCITT_GO_CLI="$(pwd)/scitt"

        # Verify binary works
        if "$SCITT_GO_CLI" --version 2>/dev/null || "$SCITT_GO_CLI" --help 2>/dev/null; then
            log_info "Go CLI verified: $SCITT_GO_CLI"
            return 0
        else
            log_warn "Go CLI built but may not be functional"
            return 0
        fi
    else
        log_error "Go build failed"
        return 1
    fi
}

# Verify TypeScript implementation
verify_typescript() {
    log_step "Verifying TypeScript SCITT implementation..."

    local repo_root=$(find_repo_root)
    local ts_impl_dir="$repo_root/scitt-typescript"

    # Try alternative locations
    if [ ! -d "$ts_impl_dir" ]; then
        ts_impl_dir="$repo_root"
        if [ ! -f "$ts_impl_dir/package.json" ]; then
            log_error "TypeScript implementation not found. Searched:"
            log_error "  - $repo_root/scitt-typescript"
            log_error "  - $repo_root"
            return 1
        fi
    fi

    log_info "Found TypeScript implementation at: $ts_impl_dir"

    # Check for CLI entry point
    local cli_path=""
    if [ -f "$ts_impl_dir/src/cli/index.ts" ]; then
        cli_path="$ts_impl_dir/src/cli/index.ts"
    elif [ -f "$ts_impl_dir/src/index.ts" ]; then
        cli_path="$ts_impl_dir/src/index.ts"
    else
        log_error "Cannot find TypeScript CLI entry point"
        return 1
    fi

    log_info "TypeScript CLI entry point: $cli_path"

    # Verify Bun is installed
    if ! command -v bun &> /dev/null; then
        log_error "Bun is not installed. Install from https://bun.sh"
        return 1
    fi

    log_info "Bun version: $(bun --version)"

    # Install dependencies if needed
    if [ ! -d "$ts_impl_dir/node_modules" ]; then
        log_info "Installing TypeScript dependencies..."
        cd "$ts_impl_dir"
        if ! bun install; then
            log_error "Failed to install TypeScript dependencies"
            return 1
        fi
    fi

    # Set CLI command
    export SCITT_TS_CLI="bun run $cli_path"
    log_info "TypeScript CLI command: $SCITT_TS_CLI"

    # Verify CLI works
    if eval "$SCITT_TS_CLI --version" 2>/dev/null || eval "$SCITT_TS_CLI --help" 2>/dev/null; then
        log_info "TypeScript CLI verified"
        return 0
    else
        log_warn "TypeScript CLI may not be fully functional"
        return 0
    fi
}

# Check binaries exist and are executable
check_binaries() {
    log_step "Verifying built binaries..."

    local go_ok=false
    local ts_ok=false

    # Check Go CLI
    if [ -n "${SCITT_GO_CLI:-}" ] && [ -x "$SCITT_GO_CLI" ]; then
        log_info "Go CLI ready: $SCITT_GO_CLI"
        go_ok=true
    else
        log_warn "Go CLI not available or not executable"
    fi

    # Check TypeScript CLI
    if [ -n "${SCITT_TS_CLI:-}" ]; then
        log_info "TypeScript CLI ready: $SCITT_TS_CLI"
        ts_ok=true
    else
        log_warn "TypeScript CLI not available"
    fi

    if [ "$go_ok" = false ] && [ "$ts_ok" = false ]; then
        log_error "Neither implementation is available"
        return 1
    fi

    log_info "Build verification complete"
    return 0
}

# Main build function
main() {
    log_info "Starting SCITT implementation build process..."

    local go_result=0
    local ts_result=0

    # Build Go (continue on failure)
    build_go || go_result=$?

    # Verify TypeScript (continue on failure)
    verify_typescript || ts_result=$?

    # Check final status
    check_binaries

    if [ $? -eq 0 ]; then
        log_info "Build process completed successfully"
        log_info "Export these variables to use the built implementations:"
        log_info "  export SCITT_GO_CLI=${SCITT_GO_CLI:-<not available>}"
        log_info "  export SCITT_TS_CLI=\"${SCITT_TS_CLI:-<not available>}\""
        return 0
    else
        log_error "Build process completed with errors"
        return 1
    fi
}

# Run if executed directly
if [ "${BASH_SOURCE[0]}" == "${0}" ]; then
    main "$@"
fi
