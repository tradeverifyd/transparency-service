package lib

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ValidateRFCCompliance validates data against RFC requirements
// Returns a list of RFC violations found
func ValidateRFCCompliance(data interface{}, rfcNumber string) []RFCViolation {
	var violations []RFCViolation

	switch rfcNumber {
	case "RFC 9052": // COSE
		violations = append(violations, validateCOSE(data)...)
	case "RFC 6962": // Certificate Transparency / Merkle Trees
		violations = append(violations, validateMerkleTree(data)...)
	case "RFC 8392": // CWT
		violations = append(violations, validateCWT(data)...)
	case "RFC 7638": // JWK Thumbprint
		violations = append(violations, validateJWKThumbprint(data)...)
	default:
		// Unknown RFC, no validation
	}

	return violations
}

// validateCOSE validates COSE Sign1 structure compliance
func validateCOSE(data interface{}) []RFCViolation {
	var violations []RFCViolation

	dataMap, ok := data.(map[string]interface{})
	if !ok {
		violations = append(violations, RFCViolation{
			RFCNumber:     "RFC 9052",
			Requirement:   "COSE Sign1 must be a CBOR map",
			Implementation: "both",
			ViolationType: "wrong_type",
			Description:   "Data is not a map structure",
		})
		return violations
	}

	// Check for required COSE Sign1 fields
	requiredFields := []string{"protected", "unprotected", "payload", "signature"}
	for _, field := range requiredFields {
		if _, exists := dataMap[field]; !exists {
			violations = append(violations, RFCViolation{
				RFCNumber:     "RFC 9052",
				Section:       "4.2",
				Requirement:   fmt.Sprintf("COSE Sign1 must contain '%s' field", field),
				Implementation: "both",
				ViolationType: "missing_field",
				Description:   fmt.Sprintf("Missing required field: %s", field),
			})
		}
	}

	return violations
}

// validateMerkleTree validates Merkle tree structure compliance with RFC 6962
func validateMerkleTree(data interface{}) []RFCViolation {
	var violations []RFCViolation

	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return violations
	}

	// Check tree size
	if treeSize, exists := dataMap["tree_size"]; exists {
		if size, ok := treeSize.(float64); ok {
			if size < 0 {
				violations = append(violations, RFCViolation{
					RFCNumber:      "RFC 6962",
					Section:        "2.1",
					Requirement:    "Tree size must be non-negative",
					Implementation: "both",
					ViolationType:  "out_of_range",
					Description:    "Tree size is negative",
					ActualValue:    size,
					ExpectedValue:  "size >= 0",
				})
			}
		}
	}

	// Validate root hash format (should be hex-encoded)
	if rootHash, exists := dataMap["root_hash"]; exists {
		if hashStr, ok := rootHash.(string); ok {
			if !isValidHex(hashStr) {
				violations = append(violations, RFCViolation{
					RFCNumber:      "RFC 6962",
					Requirement:    "Root hash should be valid hex encoding",
					Implementation: "both",
					ViolationType:  "wrong_format",
					Description:    "Root hash is not valid hexadecimal",
					ActualValue:    hashStr,
				})
			}
		}
	}

	return violations
}

// validateCWT validates CBOR Web Token compliance
func validateCWT(data interface{}) []RFCViolation {
	var violations []RFCViolation

	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return violations
	}

	// Check for standard CWT claims (iss, sub, aud, exp, nbf, iat, cti)
	// Per RFC 8392, these are optional but if present must be correct type

	// Validate issuer (iss) - must be string
	if iss, exists := dataMap["iss"]; exists {
		if _, ok := iss.(string); !ok {
			violations = append(violations, RFCViolation{
				RFCNumber:      "RFC 8392",
				Section:        "3.1.1",
				Requirement:    "Issuer (iss) claim must be a string",
				Implementation: "both",
				ViolationType:  "wrong_type",
				Description:    "iss claim is not a string",
				ActualValue:    iss,
			})
		}
	}

	// Validate subject (sub) - must be string
	if sub, exists := dataMap["sub"]; exists {
		if _, ok := sub.(string); !ok {
			violations = append(violations, RFCViolation{
				RFCNumber:      "RFC 8392",
				Section:        "3.1.2",
				Requirement:    "Subject (sub) claim must be a string",
				Implementation: "both",
				ViolationType:  "wrong_type",
				Description:    "sub claim is not a string",
				ActualValue:    sub,
			})
		}
	}

	return violations
}

// validateJWKThumbprint validates JWK thumbprint per RFC 7638
func validateJWKThumbprint(data interface{}) []RFCViolation {
	var violations []RFCViolation

	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return violations
	}

	// Check for required JWK fields for P-256 keys
	requiredFields := []string{"kty", "crv", "x", "y"}
	for _, field := range requiredFields {
		if _, exists := dataMap[field]; !exists {
			violations = append(violations, RFCViolation{
				RFCNumber:     "RFC 7638",
				Section:       "3.1",
				Requirement:   fmt.Sprintf("JWK must contain '%s' field", field),
				Implementation: "both",
				ViolationType: "missing_field",
				Description:   fmt.Sprintf("Missing required JWK field: %s", field),
			})
		}
	}

	// Validate kty
	if kty, exists := dataMap["kty"]; exists {
		if ktyStr, ok := kty.(string); ok {
			if ktyStr != "EC" {
				violations = append(violations, RFCViolation{
					RFCNumber:      "RFC 7638",
					Requirement:    "Key type (kty) must be 'EC' for elliptic curve keys",
					Implementation: "both",
					ViolationType:  "invalid_value",
					Description:    "Invalid kty value for EC key",
					ActualValue:    ktyStr,
					ExpectedValue:  "EC",
				})
			}
		}
	}

	// Validate crv for P-256
	if crv, exists := dataMap["crv"]; exists {
		if crvStr, ok := crv.(string); ok {
			if crvStr != "P-256" {
				violations = append(violations, RFCViolation{
					RFCNumber:      "RFC 7638",
					Requirement:    "Curve (crv) must be 'P-256' for ES256 keys",
					Implementation: "both",
					ViolationType:  "invalid_value",
					Description:    "Invalid crv value",
					ActualValue:    crvStr,
					ExpectedValue:  "P-256",
				})
			}
		}
	}

	return violations
}

// ValidateSnakeCase checks if JSON uses snake_case naming convention
func ValidateSnakeCase(data map[string]interface{}) []RFCViolation {
	var violations []RFCViolation

	for key := range data {
		if !isSnakeCase(key) {
			violations = append(violations, RFCViolation{
				RFCNumber:     "Project Requirement",
				Requirement:   "All JSON field names must use snake_case",
				Implementation: "both",
				ViolationType: "wrong_format",
				Description:   fmt.Sprintf("Field '%s' does not use snake_case", key),
				ActualValue:   key,
			})
		}

		// Recursively check nested objects
		if nested, ok := data[key].(map[string]interface{}); ok {
			violations = append(violations, ValidateSnakeCase(nested)...)
		}
	}

	return violations
}

// ValidateHexEncoding checks if identifiers use lowercase hex encoding
func ValidateHexEncoding(value string, fieldName string) *RFCViolation {
	if !isValidHex(value) {
		return &RFCViolation{
			RFCNumber:     "Project Requirement",
			Requirement:   "All identifiers must use lowercase hex encoding",
			Implementation: "both",
			ViolationType: "wrong_format",
			Description:   fmt.Sprintf("Field '%s' is not valid hex", fieldName),
			ActualValue:   value,
		}
	}

	// Check if uppercase hex (should be lowercase)
	if value != strings.ToLower(value) {
		return &RFCViolation{
			RFCNumber:     "Project Requirement",
			Requirement:   "Hex encoding must be lowercase",
			Implementation: "both",
			ViolationType: "wrong_format",
			Description:   fmt.Sprintf("Field '%s' contains uppercase hex", fieldName),
			ActualValue:   value,
			ExpectedValue: strings.ToLower(value),
		}
	}

	return nil
}

// isValidHex checks if a string is valid hexadecimal
func isValidHex(s string) bool {
	matched, _ := regexp.MatchString("^[0-9a-fA-F]+$", s)
	return matched
}

// isSnakeCase checks if a string follows snake_case convention
func isSnakeCase(s string) bool {
	// snake_case: lowercase letters, numbers, underscores only
	// Must not start with underscore or number
	matched, _ := regexp.MatchString("^[a-z][a-z0-9_]*$", s)
	return matched
}

// ValidateJSONStructure validates that JSON can be parsed and contains expected structure
func ValidateJSONStructure(data []byte, expectedFields []string) []RFCViolation {
	var violations []RFCViolation

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		violations = append(violations, RFCViolation{
			RFCNumber:     "RFC 8259",
			Requirement:   "Valid JSON syntax",
			Implementation: "both",
			ViolationType: "wrong_format",
			Description:   fmt.Sprintf("Invalid JSON: %v", err),
		})
		return violations
	}

	// Check for expected fields
	for _, field := range expectedFields {
		if _, exists := parsed[field]; !exists {
			violations = append(violations, RFCViolation{
				RFCNumber:     "API Contract",
				Requirement:   fmt.Sprintf("JSON must contain '%s' field", field),
				Implementation: "both",
				ViolationType: "missing_field",
				Description:   fmt.Sprintf("Missing expected field: %s", field),
			})
		}
	}

	// Validate snake_case
	violations = append(violations, ValidateSnakeCase(parsed)...)

	return violations
}
