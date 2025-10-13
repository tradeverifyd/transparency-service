package lib

import (
	"fmt"
)

// ExtractEntryID extracts entry_id from response
//
// Both implementations now use snake_case "entry_id" as an integer
func ExtractEntryID(data map[string]interface{}, impl string) (string, error) {
	// Both implementations now use snake_case "entry_id" as integer
	if id, ok := data["entry_id"].(float64); ok {
		return fmt.Sprintf("%d", int(id)), nil
	}
	return "", fmt.Errorf("entry_id not found or invalid type in %s response", impl)
}

// ExtractStatementHash extracts statement hash from response
//
// Both implementations now use snake_case "statement_hash" (when present)
func ExtractStatementHash(data map[string]interface{}, impl string) (string, bool) {
	// Both implementations now use snake_case "statement_hash"
	if hash, ok := data["statement_hash"].(string); ok {
		return hash, true
	}
	return "", false
}

// ValidateRegistrationResponse validates POST /entries response fields
//
// Returns a list of missing required fields (consistent format now)
func ValidateRegistrationResponse(data map[string]interface{}, impl string) []string {
	var missing []string

	// Both implementations now use snake_case "entry_id"
	if _, ok := data["entry_id"]; !ok {
		missing = append(missing, "entry_id")
	}

	// statement_hash is optional but recommended
	// receipt is optional but recommended

	return missing
}

// ValidateReceiptResponse validates GET /entries/{id} response fields
//
// Returns a list of missing required fields (consistent format now)
func ValidateReceiptResponse(data map[string]interface{}, impl string) []string {
	var missing []string

	// Both implementations now use snake_case
	expectedFields := []string{"entry_id", "statement_hash", "tree_size"}
	for _, field := range expectedFields {
		if _, ok := data[field]; !ok {
			missing = append(missing, field)
		}
	}

	return missing
}

// NormalizeFieldName converts between camelCase and snake_case based on implementation
//
// This helps with comparing responses that use different naming conventions
func NormalizeFieldName(field string, fromImpl string, toImpl string) string {
	// If same implementation, no conversion needed
	if fromImpl == toImpl {
		return field
	}

	// Common field mappings between Go (camelCase) and TypeScript (snake_case)
	conversions := map[string]map[string]string{
		"go_to_ts": {
			"entryId":       "entry_id",
			"statementHash": "statement_hash",
			"leafIndex":     "leaf_index",
			"treeSize":      "tree_size",
		},
		"ts_to_go": {
			"entry_id":       "entryId",
			"statement_hash": "statementHash",
			"leaf_index":     "leafIndex",
			"tree_size":      "treeSize",
		},
	}

	var conversionKey string
	if fromImpl == "go" && toImpl == "typescript" {
		conversionKey = "go_to_ts"
	} else if fromImpl == "typescript" && toImpl == "go" {
		conversionKey = "ts_to_go"
	}

	if converted, ok := conversions[conversionKey][field]; ok {
		return converted
	}

	// If no mapping found, return original
	return field
}

// CompareRegistrationResponses compares POST /entries responses
//
// Both implementations now use consistent snake_case and integer entry IDs
func CompareRegistrationResponses(goData, tsData map[string]interface{}) *ComparisonResult {
	result := &ComparisonResult{
		OutputsEquivalent: true,
		Verdict:           "equivalent",
		Differences:       []Difference{},
	}

	// Extract and compare entry IDs (both should be integers in snake_case)
	goEntryID, goErr := ExtractEntryID(goData, "go")
	tsEntryID, tsErr := ExtractEntryID(tsData, "typescript")

	if goErr != nil || tsErr != nil {
		result.OutputsEquivalent = false
		result.Verdict = "divergent"
		result.Differences = append(result.Differences, Difference{
			FieldPath:     "entry_id",
			GoValue:       goEntryID,
			TsValue:       tsEntryID,
			Severity:      "major",
			Explanation:   "Entry ID missing or invalid in one implementation",
			ViolationType: "missing_field",
		})
	}

	// Check for optional fields that may differ
	_, goHashOk := ExtractStatementHash(goData, "go")
	_, tsHashOk := ExtractStatementHash(tsData, "typescript")

	if goHashOk != tsHashOk {
		result.Differences = append(result.Differences, Difference{
			FieldPath:     "statement_hash",
			Severity:      "minor",
			Explanation:   "One implementation includes statement_hash, the other omits it (both acceptable)",
			ViolationType: "optional_field",
		})
	}

	// Check for receipt field (optional)
	_, goHasReceipt := goData["receipt"]
	_, tsHasReceipt := tsData["receipt"]

	if goHasReceipt != tsHasReceipt {
		result.Differences = append(result.Differences, Difference{
			FieldPath:     "receipt",
			Severity:      "minor",
			Explanation:   "One implementation includes receipt in POST response (both approaches valid)",
			ViolationType: "optional_field",
		})
	}

	// If only minor differences, consider it compatible
	if len(result.Differences) > 0 {
		allMinor := true
		for _, diff := range result.Differences {
			if diff.Severity != "minor" {
				allMinor = false
				break
			}
		}
		if allMinor {
			result.Verdict = "compatible"
		}
	}

	return result
}
