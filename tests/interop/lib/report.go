package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GenerateReport creates both JSON and Markdown test reports from test results
func GenerateReport(results []TestResult, outputDir string) error {
	// Generate report structure
	report := aggregateResults(results)

	// Generate JSON report
	if err := generateJSONReport(report, outputDir); err != nil {
		return fmt.Errorf("failed to generate JSON report: %w", err)
	}

	// Generate Markdown report
	if err := generateMarkdownReport(report, outputDir); err != nil {
		return fmt.Errorf("failed to generate Markdown report: %w", err)
	}

	return nil
}

// aggregateResults aggregates individual test results into a complete report
func aggregateResults(results []TestResult) *TestReport {
	report := &TestReport{
		ReportID:        fmt.Sprintf("report-%d", time.Now().Unix()),
		ExecutedAt:      time.Now().Format(time.RFC3339),
		TotalTests:      len(results),
		TestResults:     results,
		CategorySummary: generateCategorySummary(results),
		RFCCompliance:   generateRFCComplianceSummary(results),
		Performance:     generatePerformanceSummary(results),
	}

	// Count pass/fail
	for _, result := range results {
		if result.Verdict == "identical" || result.Verdict == "equivalent" {
			report.PassedTests++
		} else {
			report.FailedTests++
		}
	}

	return report
}

// generateCategorySummary creates a summary grouped by test category
func generateCategorySummary(results []TestResult) map[string]CategorySummary {
	summary := make(map[string]CategorySummary)

	for _, result := range results {
		category := result.Category
		if category == "" {
			category = "uncategorized"
		}

		cat, exists := summary[category]
		if !exists {
			cat = CategorySummary{
				Category:    category,
				TotalTests:  0,
				PassedTests: 0,
				FailedTests: 0,
			}
		}

		cat.TotalTests++
		if result.Verdict == "identical" || result.Verdict == "equivalent" {
			cat.PassedTests++
		} else {
			cat.FailedTests++
			cat.FailedTestIDs = append(cat.FailedTestIDs, result.TestID)
		}

		summary[category] = cat
	}

	return summary
}

// generateRFCComplianceSummary creates a summary of RFC compliance across tests
func generateRFCComplianceSummary(results []TestResult) map[string]RFCComplianceSummary {
	summary := make(map[string]RFCComplianceSummary)

	for _, result := range results {
		for _, rfc := range result.RFCsValidated {
			rfcSum, exists := summary[rfc]
			if !exists {
				rfcSum = RFCComplianceSummary{
					RFCNumber:       rfc,
					TotalTests:      0,
					CompliantTests:  0,
					ViolationsByType: make(map[string]int),
				}
			}

			rfcSum.TotalTests++
			if result.Comparison != nil && result.Comparison.BothRFCCompliant {
				rfcSum.CompliantTests++
			}

			// Count violation types
			if result.Comparison != nil {
				for _, diff := range result.Comparison.Differences {
					if diff.RFCReference == rfc {
						rfcSum.ViolationsByType[diff.ViolationType]++
					}
				}
			}

			summary[rfc] = rfcSum
		}
	}

	return summary
}

// generatePerformanceSummary creates a summary of performance metrics
func generatePerformanceSummary(results []TestResult) PerformanceSummary {
	summary := PerformanceSummary{
		GoImplementation: ImplementationPerformance{
			Implementation: "go",
		},
		TsImplementation: ImplementationPerformance{
			Implementation: "typescript",
		},
	}

	goCount := 0
	tsCount := 0

	for _, result := range results {
		// Try to get duration from comparison result if available
		if result.Comparison != nil {
			// Count successes based on verdict
			if result.Verdict == "identical" || result.Verdict == "equivalent" {
				summary.GoImplementation.SuccessfulRuns++
				summary.TsImplementation.SuccessfulRuns++
			} else {
				summary.GoImplementation.FailedRuns++
				summary.TsImplementation.FailedRuns++
			}
			goCount++
			tsCount++
		}
	}

	// Calculate averages
	if goCount > 0 {
		summary.GoImplementation.AverageDurationMs = summary.GoImplementation.TotalDurationMs / goCount
	}
	if tsCount > 0 {
		summary.TsImplementation.AverageDurationMs = summary.TsImplementation.TotalDurationMs / tsCount
	}

	return summary
}

// generateJSONReport writes the report to a JSON file
func generateJSONReport(report *TestReport, outputDir string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate filename with timestamp
	filename := filepath.Join(outputDir, fmt.Sprintf("test-report-%s.json", report.ReportID))

	// Marshal report to JSON
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	fmt.Printf("JSON report written to: %s\n", filename)
	return nil
}

// generateMarkdownReport writes the report to a Markdown file
func generateMarkdownReport(report *TestReport, outputDir string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate filename with timestamp
	filename := filepath.Join(outputDir, fmt.Sprintf("test-report-%s.md", report.ReportID))

	// Build Markdown content
	var md strings.Builder

	// Header
	md.WriteString("# SCITT Cross-Implementation Test Report\n\n")
	md.WriteString(fmt.Sprintf("**Report ID:** %s\n\n", report.ReportID))
	md.WriteString(fmt.Sprintf("**Executed At:** %s\n\n", report.ExecutedAt))

	// Executive Summary
	md.WriteString("## Executive Summary\n\n")
	md.WriteString(fmt.Sprintf("- **Total Tests:** %d\n", report.TotalTests))
	md.WriteString(fmt.Sprintf("- **Passed:** %d (%.1f%%)\n", report.PassedTests, float64(report.PassedTests)/float64(report.TotalTests)*100))
	md.WriteString(fmt.Sprintf("- **Failed:** %d (%.1f%%)\n\n", report.FailedTests, float64(report.FailedTests)/float64(report.TotalTests)*100))

	// Category Summary
	md.WriteString("## Test Results by Category\n\n")
	md.WriteString("| Category | Total | Passed | Failed | Pass Rate |\n")
	md.WriteString("|----------|-------|--------|--------|----------|\n")
	for category, summary := range report.CategorySummary {
		passRate := float64(summary.PassedTests) / float64(summary.TotalTests) * 100
		md.WriteString(fmt.Sprintf("| %s | %d | %d | %d | %.1f%% |\n",
			category, summary.TotalTests, summary.PassedTests, summary.FailedTests, passRate))
	}
	md.WriteString("\n")

	// RFC Compliance Summary
	md.WriteString("## RFC Compliance Summary\n\n")
	md.WriteString("| RFC | Total Tests | Compliant | Compliance Rate |\n")
	md.WriteString("|-----|-------------|-----------|----------------|\n")
	for rfc, summary := range report.RFCCompliance {
		complianceRate := float64(summary.CompliantTests) / float64(summary.TotalTests) * 100
		md.WriteString(fmt.Sprintf("| %s | %d | %d | %.1f%% |\n",
			rfc, summary.TotalTests, summary.CompliantTests, complianceRate))
	}
	md.WriteString("\n")

	// Performance Summary
	md.WriteString("## Performance Summary\n\n")
	md.WriteString("| Implementation | Avg Duration (ms) | Successful Runs | Failed Runs |\n")
	md.WriteString("|----------------|-------------------|-----------------|-------------|\n")
	md.WriteString(fmt.Sprintf("| Go | %d | %d | %d |\n",
		report.Performance.GoImplementation.AverageDurationMs,
		report.Performance.GoImplementation.SuccessfulRuns,
		report.Performance.GoImplementation.FailedRuns))
	md.WriteString(fmt.Sprintf("| TypeScript | %d | %d | %d |\n\n",
		report.Performance.TsImplementation.AverageDurationMs,
		report.Performance.TsImplementation.SuccessfulRuns,
		report.Performance.TsImplementation.FailedRuns))

	// Detailed Test Results
	md.WriteString("## Detailed Test Results\n\n")
	for _, result := range report.TestResults {
		status := "✅ PASS"
		if result.Verdict != "identical" && result.Verdict != "equivalent" {
			status = "❌ FAIL"
		}

		md.WriteString(fmt.Sprintf("### %s %s\n\n", status, result.TestName))
		md.WriteString(fmt.Sprintf("**Test ID:** %s\n\n", result.TestID))
		md.WriteString(fmt.Sprintf("**Category:** %s\n\n", result.Category))
		md.WriteString(fmt.Sprintf("**Verdict:** %s\n\n", result.Verdict))

		if result.Comparison != nil {
			md.WriteString(fmt.Sprintf("**Verdict Reason:** %s\n\n", result.Comparison.VerdictReason))

			if len(result.Comparison.Differences) > 0 {
				md.WriteString("**Differences Found:**\n\n")
				for i, diff := range result.Comparison.Differences {
					md.WriteString(fmt.Sprintf("%d. **%s** [%s]\n", i+1, diff.FieldPath, diff.Severity))
					md.WriteString(fmt.Sprintf("   - Go: `%v`\n", diff.GoValue))
					md.WriteString(fmt.Sprintf("   - TypeScript: `%v`\n", diff.TsValue))
					md.WriteString(fmt.Sprintf("   - Reason: %s\n", diff.Explanation))
					if diff.RFCReference != "" {
						md.WriteString(fmt.Sprintf("   - RFC: %s\n", diff.RFCReference))
					}
					md.WriteString("\n")
				}
			}
		}

		// Add test metadata
		if len(result.RFCsValidated) > 0 {
			md.WriteString(fmt.Sprintf("**RFCs Validated:** %s\n\n", strings.Join(result.RFCsValidated, ", ")))
		}

		md.WriteString("---\n\n")
	}

	// Write to file
	if err := os.WriteFile(filename, []byte(md.String()), 0644); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	fmt.Printf("Markdown report written to: %s\n", filename)
	return nil
}

// SaveTestResult saves a single test result to a JSON file
func SaveTestResult(result *TestResult, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	filename := filepath.Join(outputDir, fmt.Sprintf("%s.json", result.TestID))

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal test result: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write test result file: %w", err)
	}

	return nil
}

// LoadTestResults loads all test results from a directory
func LoadTestResults(inputDir string) ([]TestResult, error) {
	var results []TestResult

	// Read all JSON files in directory
	entries, err := os.ReadDir(inputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read input directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filename := filepath.Join(inputDir, entry.Name())
		data, err := os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
		}

		var result TestResult
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal file %s: %w", filename, err)
		}

		results = append(results, result)
	}

	return results, nil
}

// FormatTestSummary generates a concise summary string for a test result
func FormatTestSummary(result *TestResult) string {
	status := "PASS"
	if result.Verdict != "identical" && result.Verdict != "equivalent" {
		status = "FAIL"
	}

	summary := fmt.Sprintf("[%s] %s (%s)", status, result.TestName, result.TestID)

	if result.Comparison != nil && len(result.Comparison.Differences) > 0 {
		summary += fmt.Sprintf(" - %d difference(s)", len(result.Comparison.Differences))
	}

	return summary
}

// PrintTestSummary prints a test result summary to stdout
func PrintTestSummary(result *TestResult) {
	fmt.Println(FormatTestSummary(result))

	if result.Comparison != nil && result.Comparison.VerdictReason != "" {
		fmt.Printf("  Reason: %s\n", result.Comparison.VerdictReason)
	}

	// Print test details
	if result.Category != "" {
		fmt.Printf("  Category: %s\n", result.Category)
	}
}
