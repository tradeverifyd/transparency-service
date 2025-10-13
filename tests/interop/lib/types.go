package lib

import "time"

// TestExecutionContext represents the environment and configuration for a single test run
type TestExecutionContext struct {
	// Identification
	TestID        string    `json:"test_id"`
	TestName      string    `json:"test_name"`
	Timestamp     time.Time `json:"timestamp"`

	// Implementation metadata
	GoVersion      string `json:"go_version"`
	TsVersion      string `json:"ts_version"`
	GoBinaryPath   string `json:"go_binary_path"`
	TsCLICommand   string `json:"ts_cli_command"`

	// Environment
	GoWorkDir    string `json:"go_work_dir"`
	TsWorkDir    string `json:"ts_work_dir"`
	GoServerPort int    `json:"go_server_port"`
	TsServerPort int    `json:"ts_server_port"`

	// Test configuration
	TimeoutSeconds  int  `json:"timeout_seconds"`
	ParallelEnabled bool `json:"parallel_enabled"`
	CleanupOnPass   bool `json:"cleanup_on_pass"`
}

// ImplementationResult represents the outcome of executing an operation in one implementation
type ImplementationResult struct {
	// Implementation
	Implementation string `json:"implementation"` // "go" or "typescript"

	// Execution
	Command    []string `json:"command"`
	ExitCode   int      `json:"exit_code"`
	Stdout     string   `json:"stdout"`
	Stderr     string   `json:"stderr"`

	// Output
	OutputFormat string      `json:"output_format"` // "json", "cbor", "text", "hex"
	OutputData   interface{} `json:"output_data"`   // Parsed structured output
	OutputRaw    string      `json:"output_raw"`    // Hex-encoded raw output bytes

	// Timing
	DurationMs int `json:"duration_ms"`

	// Success
	Success bool `json:"success"`
}

// ComparisonResult represents the outcome of comparing results from both implementations
type ComparisonResult struct {
	// Equivalence
	OutputsEquivalent bool `json:"outputs_equivalent"`

	// Differences
	Differences []Difference `json:"differences,omitempty"`

	// RFC compliance
	BothRFCCompliant bool `json:"both_rfc_compliant"`

	// Verdict
	Verdict       string `json:"verdict"`        // "identical", "equivalent", "divergent", "both_invalid"
	VerdictReason string `json:"verdict_reason"` // Explanation
}

// Difference represents a specific difference between implementation outputs
type Difference struct {
	// Location
	FieldPath string `json:"field_path"` // JSON path (e.g., "$.entry_id")

	// Values
	GoValue interface{} `json:"go_value"`
	TsValue interface{} `json:"ts_value"`

	// Severity
	Severity string `json:"severity"` // "critical", "major", "minor", "acceptable"

	// Explanation
	Explanation   string `json:"explanation"`
	ViolationType string `json:"violation_type,omitempty"` // Type of violation if RFC-related
	RFCReference  string `json:"rfc_reference,omitempty"`
}

// TestResult represents the outcome of a single test case
type TestResult struct {
	// Identification
	TestID             string `json:"test_id"`
	TestName           string `json:"test_name"`
	Category           string `json:"category"`
	ExecutedAt         string `json:"executed_at"`
	ExecutionContextID string `json:"execution_context_id,omitempty"`

	// Outcome
	Status     string `json:"status,omitempty"` // "pass", "fail", "skip", "error"
	DurationMs int    `json:"duration_ms,omitempty"`

	// Results per implementation
	GoResult interface{} `json:"go_result,omitempty"` // Can be ImplementationResult or raw value
	TsResult interface{} `json:"ts_result,omitempty"` // Can be ImplementationResult or raw value

	// Comparison
	Comparison *ComparisonResult `json:"comparison,omitempty"`

	// Verdict (from comparison)
	Verdict string `json:"verdict,omitempty"` // "identical", "equivalent", "divergent", "both_invalid"

	// Diagnostics
	ErrorMessage string   `json:"error_message,omitempty"`
	StackTrace   string   `json:"stack_trace,omitempty"`
	Logs         []string `json:"logs,omitempty"`

	// Metadata
	RFCsValidated []string       `json:"rfcs_validated,omitempty"`
	RFCViolations []RFCViolation `json:"rfc_violations,omitempty"`
}

// RFCViolation represents a violation of an RFC requirement
type RFCViolation struct {
	// RFC reference
	RFCNumber   string `json:"rfc_number"`
	Section     string `json:"section,omitempty"`
	Requirement string `json:"requirement"`

	// Violation details
	Implementation  string `json:"implementation"` // "go", "typescript", or "both"
	ViolationType   string `json:"violation_type"` // "missing_field", "invalid_value", "wrong_format", etc.
	Description     string `json:"description"`

	// Evidence
	ActualValue   interface{} `json:"actual_value,omitempty"`
	ExpectedValue interface{} `json:"expected_value,omitempty"`
}

// TestReport aggregates multiple test results into a comprehensive report
type TestReport struct {
	// Identification
	ReportID   string `json:"report_id"`
	ExecutedAt string `json:"executed_at"` // RFC3339 timestamp

	// Summary
	TotalTests   int `json:"total_tests"`
	PassedTests  int `json:"passed_tests"`
	FailedTests  int `json:"failed_tests"`
	SkippedTests int `json:"skipped_tests,omitempty"`
	ErrorTests   int `json:"error_tests,omitempty"`

	// Duration
	TotalDurationMs int `json:"total_duration_ms,omitempty"`

	// Implementation versions
	GoVersion string `json:"go_version,omitempty"`
	TsVersion string `json:"ts_version,omitempty"`

	// Results
	TestResults []TestResult `json:"test_results"`

	// Analysis
	CategorySummary map[string]CategorySummary `json:"category_summary,omitempty"`
	RFCCompliance   map[string]RFCComplianceSummary `json:"rfc_compliance,omitempty"`
	Performance     PerformanceSummary        `json:"performance,omitempty"`

	// Failures
	FailedTestDetails []FailedTestDetail `json:"failed_test_details,omitempty"`
}

// CategorySummary summarizes test results for a specific category
type CategorySummary struct {
	Category       string   `json:"category"`
	TotalTests     int      `json:"total_tests"`
	PassedTests    int      `json:"passed_tests"`
	FailedTests    int      `json:"failed_tests"`
	FailedTestIDs  []string `json:"failed_test_ids,omitempty"`
}

// RFCComplianceSummary summarizes RFC compliance for a specific RFC
type RFCComplianceSummary struct {
	RFCNumber         string         `json:"rfc_number"`
	TotalTests        int            `json:"total_tests"`
	CompliantTests    int            `json:"compliant_tests"`
	ViolationsByType  map[string]int `json:"violations_by_type,omitempty"`
}

// PerformanceSummary summarizes performance metrics across tests
type PerformanceSummary struct {
	GoImplementation ImplementationPerformance `json:"go_implementation"`
	TsImplementation ImplementationPerformance `json:"ts_implementation"`
}

// ImplementationPerformance summarizes performance for one implementation
type ImplementationPerformance struct {
	Implementation    string `json:"implementation"`
	TotalDurationMs   int    `json:"total_duration_ms"`
	AverageDurationMs int    `json:"average_duration_ms"`
	SuccessfulRuns    int    `json:"successful_runs"`
	FailedRuns        int    `json:"failed_runs"`
}

// FailedTestDetail provides detailed information about a failed test
type FailedTestDetail struct {
	TestName       string         `json:"test_name"`
	ErrorSummary   string         `json:"error_summary"`
	ExpectedOutput string         `json:"expected_output,omitempty"`
	ActualOutput   string         `json:"actual_output,omitempty"`
	Diff           string         `json:"diff,omitempty"`
	RFCViolations  []RFCViolation `json:"rfc_violations,omitempty"`
}
