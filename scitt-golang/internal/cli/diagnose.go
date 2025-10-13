package cli

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/spf13/cobra"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/cose"
)

// NewDiagnoseCommand creates the diagnose command
func NewDiagnoseCommand() *cobra.Command {
	var outputFile string

	cmd := &cobra.Command{
		Use:   "diagnose <file.cbor>",
		Short: "Diagnose CBOR files with extended diagnostic notation",
		Long: `Produces a markdown summary of CBOR objects including extended diagnostic notation.

This command recognizes and pretty prints:
  - COSE Keys (with algorithm and curve information)
  - COSE Sign1 structures (with protected/unprotected headers)
  - Generic CBOR objects

The output includes:
  - Structure type detection
  - Extended diagnostic notation
  - Decoded header information
  - Hex dumps of binary data`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiagnose(args[0], outputFile)
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file (default: stdout)")

	return cmd
}

// runDiagnose performs the diagnose operation
func runDiagnose(inputFile, outputFile string) error {
	// Read input file
	rawBytes, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Decode CBOR
	var data interface{}
	if err := cbor.Unmarshal(rawBytes, &data); err != nil {
		return fmt.Errorf("failed to parse CBOR: %w", err)
	}

	// Generate markdown report
	report := generateMarkdownReport(data, inputFile, rawBytes)

	// Output
	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(report), 0644); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		fmt.Printf("Diagnostic report written to: %s\n", outputFile)
	} else {
		fmt.Print(report)
	}

	return nil
}

// generateMarkdownReport creates a comprehensive markdown report
func generateMarkdownReport(data interface{}, filename string, rawBytes []byte) string {
	var buf bytes.Buffer

	// Header
	buf.WriteString("# CBOR Diagnostic Report\n\n")
	buf.WriteString(fmt.Sprintf("**File:** `%s`\n\n", filename))
	buf.WriteString(fmt.Sprintf("**Size:** %d bytes\n\n", len(rawBytes)))
	buf.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().UTC().Format(time.RFC3339)))
	buf.WriteString("---\n\n")

	// Structure Analysis
	buf.WriteString("## Structure Analysis\n\n")

	// Detect structure type
	if isCoseKey(data) {
		buf.WriteString("**Detected:** COSE Key\n\n")
		buf.WriteString(prettyPrintCoseKey(data))
	} else if isCoseSign1(data) {
		buf.WriteString("**Detected:** COSE Sign1\n\n")
		buf.WriteString(prettyPrintCoseSign1(data))
	} else {
		buf.WriteString("**Type:** Generic CBOR object\n\n")
		buf.WriteString("### Extended Diagnostic Notation\n\n")
		buf.WriteString("```cbor-diag\n")
		buf.WriteString(toExtendedDiagnostic(data, 0))
		buf.WriteString("\n```\n\n")
	}

	// Raw CBOR Data
	buf.WriteString("## Raw CBOR Data\n\n")
	buf.WriteString("```\n")
	buf.WriteString(formatHex(rawBytes, 256))
	buf.WriteString("\n```\n\n")

	return buf.String()
}

// isCoseKey checks if data is a COSE Key structure
func isCoseKey(data interface{}) bool {
	m, ok := data.(map[interface{}]interface{})
	if !ok {
		return false
	}
	// COSE Keys must have kty (label 1)
	// Try both int64 and int
	if _, hasKty := m[int64(1)]; hasKty {
		return true
	}
	if _, hasKty := m[int(1)]; hasKty {
		return true
	}
	if _, hasKty := m[uint64(1)]; hasKty {
		return true
	}
	return false
}

// isCoseSign1 checks if data is a COSE Sign1 structure
func isCoseSign1(data interface{}) bool {
	arr, ok := data.([]interface{})
	if !ok {
		return false
	}
	// COSE_Sign1 is a 4-element array
	return len(arr) == 4
}

// prettyPrintCoseKey formats a COSE Key
func prettyPrintCoseKey(data interface{}) string {
	m := data.(map[interface{}]interface{})
	var buf bytes.Buffer

	buf.WriteString("### COSE Key\n\n")
	buf.WriteString("```\n")

	// Helper to get integer value from map with flexible key types
	getInt := func(label int) (int, bool) {
		if v, ok := m[int64(label)]; ok {
			if i, ok := v.(int64); ok {
				return int(i), true
			}
			if i, ok := v.(uint64); ok {
				return int(i), true
			}
		}
		if v, ok := m[label]; ok {
			if i, ok := v.(int64); ok {
				return int(i), true
			}
			if i, ok := v.(uint64); ok {
				return int(i), true
			}
		}
		if v, ok := m[uint64(label)]; ok {
			if i, ok := v.(int64); ok {
				return int(i), true
			}
			if i, ok := v.(uint64); ok {
				return int(i), true
			}
		}
		return 0, false
	}

	// Helper to get bytes value from map
	getBytes := func(label int) ([]byte, bool) {
		if v, ok := m[int64(label)].([]byte); ok {
			return v, true
		}
		if v, ok := m[label].([]byte); ok {
			return v, true
		}
		if v, ok := m[uint64(label)].([]byte); ok {
			return v, true
		}
		return nil, false
	}

	// kty (1)
	if kty, ok := getInt(1); ok {
		buf.WriteString(fmt.Sprintf("kty (1): %s\n", getKeyTypeName(kty)))
	}

	// kid (2)
	if kid, ok := getBytes(2); ok {
		buf.WriteString(fmt.Sprintf("kid (2): %s\n", formatHex(kid, 32)))
	}

	// alg (3)
	if alg, ok := getInt(3); ok {
		buf.WriteString(fmt.Sprintf("alg (3): %s\n", getAlgorithmName(alg)))
	}

	// crv (-1)
	if crv, ok := getInt(-1); ok {
		buf.WriteString(fmt.Sprintf("crv (-1): %s\n", getCurveName(crv)))
	}

	// x (-2)
	if x, ok := getBytes(-2); ok {
		buf.WriteString(fmt.Sprintf("x (-2): %s\n", formatHex(x, 32)))
	}

	// y (-3)
	if y, ok := getBytes(-3); ok {
		buf.WriteString(fmt.Sprintf("y (-3): %s\n", formatHex(y, 32)))
	}

	// d (-4) - private key
	if d, ok := getBytes(-4); ok {
		buf.WriteString(fmt.Sprintf("d (-4): [PRIVATE KEY - %d bytes]\n", len(d)))
	}

	buf.WriteString("```\n\n")

	// Extended diagnostic notation
	buf.WriteString("#### Extended Diagnostic Notation\n\n")
	buf.WriteString("```cbor-diag\n")
	buf.WriteString(toExtendedDiagnostic(data, 0))
	buf.WriteString("\n```\n\n")

	return buf.String()
}

// prettyPrintCoseSign1 formats a COSE Sign1 structure
func prettyPrintCoseSign1(data interface{}) string {
	arr := data.([]interface{})
	var buf bytes.Buffer

	buf.WriteString("### COSE Sign1\n\n")

	// Extract components
	protectedBytes, _ := arr[0].([]byte)
	unprotected, _ := arr[1].(map[interface{}]interface{})
	payload, _ := arr[2].([]byte) // Can be nil for detached
	signature, _ := arr[3].([]byte)

	// Decode protected header
	var protected map[interface{}]interface{}
	if len(protectedBytes) > 0 {
		cbor.Unmarshal(protectedBytes, &protected)
	}

	// Protected Header
	buf.WriteString("#### Protected Header\n\n")
	buf.WriteString("```\n")
	if len(protected) == 0 {
		buf.WriteString("(empty)\n")
	} else {
		for label, value := range protected {
			labelInt, _ := label.(int64)
			labelName := getHeaderLabelName(int(labelInt))
			buf.WriteString(formatHeaderValue(labelName, int(labelInt), value))
		}
	}
	buf.WriteString("```\n\n")

	// Unprotected Header
	buf.WriteString("#### Unprotected Header\n\n")
	buf.WriteString("```\n")
	if len(unprotected) == 0 {
		buf.WriteString("(empty)\n")
	} else {
		for label, value := range unprotected {
			labelInt, _ := label.(int64)
			labelName := getHeaderLabelName(int(labelInt))
			buf.WriteString(formatHeaderValue(labelName, int(labelInt), value))
		}
	}
	buf.WriteString("```\n\n")

	// Payload
	buf.WriteString("#### Payload\n\n")
	buf.WriteString("```\n")
	if payload == nil {
		buf.WriteString("detached (null)\n")
	} else {
		buf.WriteString(formatHex(payload, 64))
		buf.WriteString("\n")
	}
	buf.WriteString("```\n\n")

	// Signature
	buf.WriteString("#### Signature\n\n")
	buf.WriteString("```\n")
	if signature != nil {
		buf.WriteString(formatHex(signature, 32))
		buf.WriteString(fmt.Sprintf("\nLength: %d bytes\n", len(signature)))
	}
	buf.WriteString("```\n\n")

	// Extended diagnostic notation
	buf.WriteString("#### Extended Diagnostic Notation\n\n")
	buf.WriteString("```cbor-diag\n")
	buf.WriteString("/ COSE_Sign1 / 18([\n")
	buf.WriteString(fmt.Sprintf("  / protected / %s,\n", toExtendedDiagnostic(protectedBytes, 1)))
	buf.WriteString(fmt.Sprintf("  / unprotected / %s,\n", toExtendedDiagnostic(unprotected, 1)))
	buf.WriteString(fmt.Sprintf("  / payload / %s,\n", toExtendedDiagnostic(payload, 1)))
	buf.WriteString(fmt.Sprintf("  / signature / %s\n", toExtendedDiagnostic(signature, 1)))
	buf.WriteString("])\n")
	buf.WriteString("```\n\n")

	return buf.String()
}

// formatHeaderValue formats a header label-value pair
func formatHeaderValue(labelName string, labelInt int, value interface{}) string {
	switch v := value.(type) {
	case []byte:
		return fmt.Sprintf("%s (%d): %s\n", labelName, labelInt, formatHex(v, 16))
	case int64:
		if labelInt == cose.HeaderLabelAlg {
			return fmt.Sprintf("%s (%d): %s\n", labelName, labelInt, getAlgorithmName(int(v)))
		}
		return fmt.Sprintf("%s (%d): %d\n", labelName, labelInt, v)
	case string:
		return fmt.Sprintf("%s (%d): %s\n", labelName, labelInt, v)
	case map[interface{}]interface{}:
		return fmt.Sprintf("%s (%d): %s\n", labelName, labelInt, toExtendedDiagnostic(v, 0))
	default:
		return fmt.Sprintf("%s (%d): %v\n", labelName, labelInt, v)
	}
}

// toExtendedDiagnostic converts CBOR data to extended diagnostic notation
func toExtendedDiagnostic(value interface{}, indent int) string {
	spaces := ""
	for i := 0; i < indent; i++ {
		spaces += "  "
	}

	switch v := value.(type) {
	case nil:
		return "null"
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int, int64, uint, uint64:
		return fmt.Sprintf("%v", v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	case string:
		return fmt.Sprintf("\"%s\"", v)
	case []byte:
		return fmt.Sprintf("h'%s'", hex.EncodeToString(v))
	case []interface{}:
		if len(v) == 0 {
			return "[]"
		}
		var buf bytes.Buffer
		buf.WriteString("[\n")
		for i, item := range v {
			buf.WriteString(spaces + "  " + toExtendedDiagnostic(item, indent+1))
			if i < len(v)-1 {
				buf.WriteString(",")
			}
			buf.WriteString("\n")
		}
		buf.WriteString(spaces + "]")
		return buf.String()
	case map[interface{}]interface{}:
		if len(v) == 0 {
			return "{}"
		}
		var buf bytes.Buffer
		buf.WriteString("{\n")
		first := true
		for key, val := range v {
			if !first {
				buf.WriteString(",\n")
			}
			first = false
			keyStr := fmt.Sprintf("%v", key)
			buf.WriteString(fmt.Sprintf("%s  %s: %s", spaces, keyStr, toExtendedDiagnostic(val, indent+1)))
		}
		buf.WriteString("\n" + spaces + "}")
		return buf.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// formatHex formats bytes as hex with spaces
func formatHex(data []byte, maxBytes int) string {
	if len(data) <= maxBytes {
		return hexWithSpaces(data)
	}
	preview := hexWithSpaces(data[:maxBytes])
	return fmt.Sprintf("%s ... (%d bytes total)", preview, len(data))
}

// hexWithSpaces converts bytes to hex string with spaces
func hexWithSpaces(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	var buf bytes.Buffer
	for i, b := range data {
		if i > 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(fmt.Sprintf("%02x", b))
	}
	return buf.String()
}

// getKeyTypeName returns the name of a COSE key type
func getKeyTypeName(kty int) string {
	switch kty {
	case 1:
		return "OKP"
	case 2:
		return "EC2"
	case 3:
		return "RSA"
	case 4:
		return "Symmetric"
	default:
		return fmt.Sprintf("Unknown (%d)", kty)
	}
}

// getCurveName returns the name of a COSE elliptic curve
func getCurveName(crv int) string {
	switch crv {
	case 1:
		return "P-256"
	case 2:
		return "P-384"
	case 3:
		return "P-521"
	case 4:
		return "X25519"
	case 5:
		return "X448"
	case 6:
		return "Ed25519"
	case 7:
		return "Ed448"
	default:
		return fmt.Sprintf("Unknown (%d)", crv)
	}
}

// getAlgorithmName returns the name of a COSE algorithm
func getAlgorithmName(alg int) string {
	switch alg {
	case -7:
		return "ES256"
	case -8:
		return "EdDSA"
	case -35:
		return "ES384"
	case -36:
		return "ES512"
	case -37:
		return "PS256"
	case -38:
		return "PS384"
	case -39:
		return "PS512"
	case -16:
		return "SHA-256"
	case -43:
		return "SHA-384"
	case -44:
		return "SHA-512"
	default:
		return fmt.Sprintf("Unknown (%d)", alg)
	}
}

// getHeaderLabelName returns the name of a COSE header label
func getHeaderLabelName(label int) string {
	switch label {
	case cose.HeaderLabelAlg:
		return "alg"
	case cose.HeaderLabelCrit:
		return "crit"
	case cose.HeaderLabelContentType:
		return "content_type"
	case cose.HeaderLabelKid:
		return "kid"
	case cose.HeaderLabelIV:
		return "iv"
	case cose.HeaderLabelPartialIV:
		return "partial_iv"
	case cose.HeaderLabelCounterSig:
		return "counter_sig"
	case cose.HeaderLabelCWTClaims:
		return "cwt_claims"
	case cose.HeaderLabelTyp:
		return "typ"
	case cose.HeaderLabelIss:
		return "iss"
	case cose.HeaderLabelSub:
		return "sub"
	case cose.HeaderLabelPayloadHashAlg:
		return "payload_hash_alg"
	case cose.HeaderLabelPayloadPreimageContentType:
		return "payload_preimage_content_type"
	case cose.HeaderLabelPayloadLocation:
		return "payload_location"
	case cose.HeaderLabelVerifiableDataStructure:
		return "vds"
	case cose.HeaderLabelVerifiableDataProof:
		return "vdp"
	case cose.HeaderLabelReceipts:
		return "receipts"
	default:
		return fmt.Sprintf("label_%d", label)
	}
}
