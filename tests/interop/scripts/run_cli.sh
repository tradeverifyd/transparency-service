#!/bin/bash
# Wrapper script to run either Go or TypeScript SCITT CLI
# Handles logging, output capture, and error handling

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
IMPL=""
WORK_DIR=""
TIMEOUT=30
LOG_FILE=""
CAPTURE_OUTPUT=false
OUTPUT_FILE=""

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1" >&2
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

log_debug() {
    if [ "${DEBUG:-0}" = "1" ]; then
        echo -e "${BLUE}[DEBUG]${NC} $1" >&2
    fi
}

usage() {
    cat <<EOF
Usage: $0 -i IMPL [OPTIONS] -- COMMAND [ARGS...]

Run SCITT CLI with logging and output capture.

OPTIONS:
  -i IMPL          Implementation to run: 'go' or 'typescript' (required)
  -w DIR           Working directory for command execution
  -t SECONDS       Timeout in seconds (default: 30)
  -l FILE          Log file for command output
  -c               Capture output to stdout
  -o FILE          Save output to file
  -h               Show this help message

ENVIRONMENT VARIABLES:
  SCITT_GO_CLI     Path to Go SCITT binary
  SCITT_TS_CLI     Command to run TypeScript CLI
  DEBUG            Set to 1 to enable debug logging

EXAMPLES:
  # Run Go CLI to register a statement
  $0 -i go -w /tmp/test -- register --payload test.json

  # Run TypeScript CLI with output capture
  $0 -i typescript -c -o output.txt -- verify --entry-id abc123

  # Run with timeout and logging
  $0 -i go -t 60 -l /tmp/test.log -- server start --port 8080
EOF
}

# Parse command line arguments
parse_args() {
    while getopts "i:w:t:l:co:h" opt; do
        case $opt in
            i) IMPL="$OPTARG" ;;
            w) WORK_DIR="$OPTARG" ;;
            t) TIMEOUT="$OPTARG" ;;
            l) LOG_FILE="$OPTARG" ;;
            c) CAPTURE_OUTPUT=true ;;
            o) OUTPUT_FILE="$OPTARG" ;;
            h) usage; exit 0 ;;
            ?) usage; exit 1 ;;
        esac
    done

    shift $((OPTIND - 1))

    # Remaining arguments are the command
    CLI_ARGS=("$@")

    # Validate required arguments
    if [ -z "$IMPL" ]; then
        log_error "Implementation (-i) is required"
        usage
        exit 1
    fi

    if [ ${#CLI_ARGS[@]} -eq 0 ]; then
        log_error "No command specified"
        usage
        exit 1
    fi
}

# Get CLI command based on implementation
get_cli_command() {
    case "$IMPL" in
        go)
            if [ -z "${SCITT_GO_CLI:-}" ]; then
                log_error "SCITT_GO_CLI environment variable not set"
                exit 1
            fi
            if [ ! -x "$SCITT_GO_CLI" ]; then
                log_error "Go CLI not executable: $SCITT_GO_CLI"
                exit 1
            fi
            echo "$SCITT_GO_CLI"
            ;;
        typescript|ts)
            if [ -z "${SCITT_TS_CLI:-}" ]; then
                log_error "SCITT_TS_CLI environment variable not set"
                exit 1
            fi
            echo "$SCITT_TS_CLI"
            ;;
        *)
            log_error "Unknown implementation: $IMPL (must be 'go' or 'typescript')"
            exit 1
            ;;
    esac
}

# Run the CLI command with timeout
run_with_timeout() {
    local cmd="$1"
    shift
    local args=("$@")

    log_debug "Running command: $cmd ${args[*]}"
    log_debug "Working directory: ${WORK_DIR:-.}"
    log_debug "Timeout: ${TIMEOUT}s"

    # Build full command
    local full_cmd=""
    if [ -n "$WORK_DIR" ]; then
        cd "$WORK_DIR"
    fi

    # For TypeScript, we need to handle "bun run ..." specially
    if [ "$IMPL" = "typescript" ] || [ "$IMPL" = "ts" ]; then
        # SCITT_TS_CLI is like "bun run path/to/cli.ts"
        full_cmd="$cmd ${args[*]}"
    else
        full_cmd="$cmd ${args[*]}"
    fi

    # Create temporary files for output
    local tmp_stdout=$(mktemp)
    local tmp_stderr=$(mktemp)
    local tmp_combined=$(mktemp)

    # Run command with timeout
    local exit_code=0
    local start_time=$(date +%s)

    if timeout "$TIMEOUT" bash -c "$full_cmd" > "$tmp_stdout" 2> "$tmp_stderr"; then
        exit_code=0
    else
        exit_code=$?
        if [ $exit_code -eq 124 ]; then
            log_error "Command timed out after ${TIMEOUT}s"
        fi
    fi

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    # Combine stdout and stderr for logging
    cat "$tmp_stdout" "$tmp_stderr" > "$tmp_combined"

    # Handle output
    if [ -n "$LOG_FILE" ]; then
        log_debug "Writing output to log: $LOG_FILE"
        cat "$tmp_combined" >> "$LOG_FILE"
    fi

    if [ "$CAPTURE_OUTPUT" = true ]; then
        cat "$tmp_stdout"
    fi

    if [ -n "$OUTPUT_FILE" ]; then
        log_debug "Writing output to file: $OUTPUT_FILE"
        cat "$tmp_stdout" > "$OUTPUT_FILE"
    fi

    # Log summary
    log_debug "Command completed with exit code $exit_code in ${duration}s"

    # Cleanup
    rm -f "$tmp_stdout" "$tmp_stderr" "$tmp_combined"

    return $exit_code
}

# Main execution
main() {
    parse_args "$@"

    local cli_cmd=$(get_cli_command)

    log_debug "Implementation: $IMPL"
    log_debug "CLI command: $cli_cmd"

    run_with_timeout "$cli_cmd" "${CLI_ARGS[@]}"
    exit $?
}

# Run if executed directly
if [ "${BASH_SOURCE[0]}" == "${0}" ]; then
    main "$@"
fi
