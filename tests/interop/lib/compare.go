package lib

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// CompareJSON performs semantic deep equality comparison of two JSON byte arrays
// Returns whether they are equivalent, list of differences, and any error
func CompareJSON(a, b []byte) (equal bool, diffs []Difference, err error) {
	// Parse both JSON documents
	var objA, objB interface{}

	if err := json.Unmarshal(a, &objA); err != nil {
		return false, nil, fmt.Errorf("failed to parse first JSON: %w", err)
	}

	if err := json.Unmarshal(b, &objB); err != nil {
		return false, nil, fmt.Errorf("failed to parse second JSON: %w", err)
	}

	// Perform deep comparison
	diffs = compareValues("$", objA, objB)

	return len(diffs) == 0, diffs, nil
}

// compareValues recursively compares two values and returns differences
func compareValues(path string, a, b interface{}) []Difference {
	var diffs []Difference

	// Handle nil cases
	if a == nil && b == nil {
		return nil
	}
	if a == nil {
		diffs = append(diffs, Difference{
			FieldPath:   path,
			GoValue:     a,
			TsValue:     b,
			Severity:    "major",
			Explanation: "Go value is null, TypeScript value is not",
		})
		return diffs
	}
	if b == nil {
		diffs = append(diffs, Difference{
			FieldPath:   path,
			GoValue:     a,
			TsValue:     b,
			Severity:    "major",
			Explanation: "TypeScript value is null, Go value is not",
		})
		return diffs
	}

	// Get types
	typeA := reflect.TypeOf(a)
	typeB := reflect.TypeOf(b)

	// Handle different types
	if typeA.Kind() != typeB.Kind() {
		// Special case: numeric type differences (int vs float with same value)
		if isNumeric(a) && isNumeric(b) {
			if numericEqual(a, b) {
				return nil // Acceptable difference
			}
		}

		diffs = append(diffs, Difference{
			FieldPath:   path,
			GoValue:     a,
			TsValue:     b,
			Severity:    "major",
			Explanation: fmt.Sprintf("Type mismatch: %T vs %T", a, b),
		})
		return diffs
	}

	// Compare based on type
	switch aVal := a.(type) {
	case map[string]interface{}:
		bVal := b.(map[string]interface{})
		diffs = append(diffs, compareObjects(path, aVal, bVal)...)

	case []interface{}:
		bVal := b.([]interface{})
		diffs = append(diffs, compareArrays(path, aVal, bVal)...)

	case string, bool:
		if !reflect.DeepEqual(a, b) {
			diffs = append(diffs, Difference{
				FieldPath:   path,
				GoValue:     a,
				TsValue:     b,
				Severity:    "major",
				Explanation: fmt.Sprintf("Value mismatch: %v != %v", a, b),
			})
		}

	case float64:
		bVal := b.(float64)
		if !floatEqual(aVal, bVal) {
			diffs = append(diffs, Difference{
				FieldPath:   path,
				GoValue:     a,
				TsValue:     b,
				Severity:    "minor",
				Explanation: fmt.Sprintf("Numeric precision difference: %v vs %v", a, b),
			})
		}
	}

	return diffs
}

// compareObjects compares two JSON objects (maps)
func compareObjects(path string, a, b map[string]interface{}) []Difference {
	var diffs []Difference

	// Check for keys in a but not in b
	for key := range a {
		if _, exists := b[key]; !exists {
			diffs = append(diffs, Difference{
				FieldPath:   fmt.Sprintf("%s.%s", path, key),
				GoValue:     a[key],
				TsValue:     nil,
				Severity:    "major",
				Explanation: "Field present in Go output but missing in TypeScript output",
			})
		}
	}

	// Check for keys in b but not in a
	for key := range b {
		if _, exists := a[key]; !exists {
			diffs = append(diffs, Difference{
				FieldPath:   fmt.Sprintf("%s.%s", path, key),
				GoValue:     nil,
				TsValue:     b[key],
				Severity:    "major",
				Explanation: "Field present in TypeScript output but missing in Go output",
			})
		}
	}

	// Compare common keys
	for key := range a {
		if valB, exists := b[key]; exists {
			fieldPath := fmt.Sprintf("%s.%s", path, key)
			diffs = append(diffs, compareValues(fieldPath, a[key], valB)...)
		}
	}

	return diffs
}

// compareArrays compares two JSON arrays
func compareArrays(path string, a, b []interface{}) []Difference {
	var diffs []Difference

	if len(a) != len(b) {
		diffs = append(diffs, Difference{
			FieldPath:   path,
			GoValue:     len(a),
			TsValue:     len(b),
			Severity:    "major",
			Explanation: fmt.Sprintf("Array length mismatch: %d vs %d", len(a), len(b)),
		})
		return diffs
	}

	// Compare each element
	for i := range a {
		elementPath := fmt.Sprintf("%s[%d]", path, i)
		diffs = append(diffs, compareValues(elementPath, a[i], b[i])...)
	}

	return diffs
}

// isNumeric checks if a value is a numeric type
func isNumeric(v interface{}) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return true
	default:
		return false
	}
}

// numericEqual checks if two numeric values are equal
func numericEqual(a, b interface{}) bool {
	aFloat := toFloat64(a)
	bFloat := toFloat64(b)
	return floatEqual(aFloat, bFloat)
}

// toFloat64 converts a numeric value to float64
func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case float64:
		return val
	case float32:
		return float64(val)
	default:
		return 0
	}
}

// floatEqual checks if two floats are equal within a small epsilon
func floatEqual(a, b float64) bool {
	const epsilon = 1e-9
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < epsilon
}

// CompareOutputs compares two ImplementationResult objects and generates a ComparisonResult
func CompareOutputs(goResult, tsResult *ImplementationResult) *ComparisonResult {
	// Parse JSON outputs if available
	var diffs []Difference
	outputsEquivalent := true

	if goResult.OutputFormat == "json" && tsResult.OutputFormat == "json" {
		equal, jsonDiffs, err := CompareJSON([]byte(goResult.Stdout), []byte(tsResult.Stdout))
		if err != nil {
			return &ComparisonResult{
				OutputsEquivalent: false,
				Verdict:           "error",
				VerdictReason:     fmt.Sprintf("Failed to compare JSON outputs: %v", err),
			}
		}
		outputsEquivalent = equal
		diffs = jsonDiffs
	} else {
		// String comparison for non-JSON outputs
		if goResult.Stdout != tsResult.Stdout {
			outputsEquivalent = false
			diffs = append(diffs, Difference{
				FieldPath:   "$.stdout",
				GoValue:     goResult.Stdout,
				TsValue:     tsResult.Stdout,
				Severity:    "major",
				Explanation: "Output strings do not match",
			})
		}
	}

	// Determine verdict
	verdict := determineVerdict(outputsEquivalent, diffs, goResult.Success, tsResult.Success)

	// Count critical differences
	criticalDiffs := 0
	for _, diff := range diffs {
		if diff.Severity == "critical" {
			criticalDiffs++
		}
	}

	verdictReason := generateVerdictReason(verdict, len(diffs), criticalDiffs)

	return &ComparisonResult{
		OutputsEquivalent: outputsEquivalent,
		Differences:       diffs,
		BothRFCCompliant:  true, // TODO: Implement RFC validation
		Verdict:           verdict,
		VerdictReason:     verdictReason,
	}
}

// determineVerdict determines the comparison verdict based on differences
func determineVerdict(outputsEquivalent bool, diffs []Difference, goSuccess, tsSuccess bool) string {
	if !goSuccess && !tsSuccess {
		return "both_invalid"
	}

	if outputsEquivalent {
		return "identical"
	}

	// Check if all differences are acceptable
	allAcceptable := true
	for _, diff := range diffs {
		if diff.Severity != "acceptable" && diff.Severity != "minor" {
			allAcceptable = false
			break
		}
	}

	if allAcceptable {
		return "equivalent"
	}

	return "divergent"
}

// generateVerdictReason generates a human-readable explanation of the verdict
func generateVerdictReason(verdict string, totalDiffs, criticalDiffs int) string {
	switch verdict {
	case "identical":
		return "Both implementations produced identical outputs"
	case "equivalent":
		return fmt.Sprintf("Outputs are semantically equivalent (%d minor differences)", totalDiffs)
	case "divergent":
		return fmt.Sprintf("Outputs diverge significantly (%d differences, %d critical)", totalDiffs, criticalDiffs)
	case "both_invalid":
		return "Both implementations failed to produce valid output"
	default:
		return "Unknown verdict"
	}
}

// FormatDifferences formats a list of differences as a human-readable string
func FormatDifferences(diffs []Difference) string {
	if len(diffs) == 0 {
		return "No differences found"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d difference(s):\n", len(diffs)))

	for i, diff := range diffs {
		sb.WriteString(fmt.Sprintf("\n%d. %s [%s]\n", i+1, diff.FieldPath, diff.Severity))
		sb.WriteString(fmt.Sprintf("   Go:         %v\n", diff.GoValue))
		sb.WriteString(fmt.Sprintf("   TypeScript: %v\n", diff.TsValue))
		sb.WriteString(fmt.Sprintf("   Reason:     %s\n", diff.Explanation))
		if diff.RFCReference != "" {
			sb.WriteString(fmt.Sprintf("   RFC:        %s\n", diff.RFCReference))
		}
	}

	return sb.String()
}
