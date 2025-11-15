package main

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/fxamacker/cbor/v2"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <cbor-file>\n", os.Args[0])
		os.Exit(1)
	}

	filename := os.Args[1]
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Decode CBOR to generic interface
	var decoded interface{}
	if err := cbor.Unmarshal(data, &decoded); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding CBOR: %v\n", err)
		os.Exit(1)
	}

	// Pretty print as EDN with structure labels
	printCOSESign1(decoded)
	fmt.Println()
}

func printCOSESign1(v interface{}) {
	arr, ok := v.([]interface{})
	if !ok || len(arr) != 4 {
		fmt.Println("Error: Not a valid COSE_Sign1 structure")
		return
	}

	fmt.Println(";; COSE_Sign1 Structure")
	fmt.Println("{:protected")
	fmt.Print("   ")
	printProtectedHeaders(arr[0])
	fmt.Println()

	fmt.Println(" :unprotected")
	fmt.Print("   ")
	printUnprotectedHeaders(arr[1])
	fmt.Println()

	fmt.Println(" :payload")
	fmt.Print("   ")
	printEDN(arr[2], 2)
	fmt.Println()

	fmt.Println(" :signature")
	fmt.Print("   ")
	printEDN(arr[3], 2)
	fmt.Println("}")
}

func printProtectedHeaders(v interface{}) {
	protected, ok := v.([]byte)
	if !ok {
		fmt.Print("nil")
		return
	}

	// Decode the protected headers
	var headers map[interface{}]interface{}
	if err := cbor.Unmarshal(protected, &headers); err != nil {
		// If can't decode, just show as hex
		fmt.Printf("#h\"%s\"", hex.EncodeToString(protected))
		return
	}

	fmt.Println("{")
	for k, val := range headers {
		fmt.Print("    ")
		printHeaderLabel(k)
		fmt.Print(" ")
		if k64, ok := k.(int64); ok && k64 == 15 {
			printCWTClaims(val)
		} else if k64, ok := k.(uint64); ok && k64 == 15 {
			printCWTClaims(val)
		} else {
			printEDN(val, 3)
		}
		fmt.Println()
	}
	fmt.Print("   }")
}

func printUnprotectedHeaders(v interface{}) {
	headers, ok := v.(map[interface{}]interface{})
	if !ok {
		fmt.Print("{}")
		return
	}

	fmt.Println("{")
	for k, val := range headers {
		fmt.Print("    ")
		printHeaderLabel(k)
		fmt.Print(" ")
		if k64, ok := k.(int64); ok && k64 == 396 {
			printVDP(val)
		} else if k64, ok := k.(uint64); ok && k64 == 396 {
			printVDP(val)
		} else {
			printEDN(val, 3)
		}
		fmt.Println()
	}
	fmt.Print("   }")
}

func printHeaderLabel(k interface{}) {
	labels := map[int64]string{
		1:   ":alg",
		4:   ":kid",
		15:  ":cwt-claims",
		396: ":vdp",
	}

	switch v := k.(type) {
	case int64:
		if label, ok := labels[v]; ok {
			fmt.Print(label)
		} else {
			fmt.Printf("%d", v)
		}
	case uint64:
		if label, ok := labels[int64(v)]; ok {
			fmt.Print(label)
		} else {
			fmt.Printf("%d", v)
		}
	default:
		printEDN(k, 0)
	}
}

func printCWTClaims(v interface{}) {
	claims, ok := v.(map[interface{}]interface{})
	if !ok {
		printEDN(v, 3)
		return
	}

	fmt.Println("{")
	for k, val := range claims {
		fmt.Print("     ")
		printCWTClaimLabel(k)
		fmt.Print(" ")
		printEDN(val, 4)
		fmt.Println()
	}
	fmt.Print("    }")
}

func printCWTClaimLabel(k interface{}) {
	labels := map[int64]string{
		1: ":iss",
		2: ":sub",
		3: ":aud",
		4: ":exp",
		5: ":nbf",
		6: ":iat",
	}

	switch v := k.(type) {
	case int64:
		if label, ok := labels[v]; ok {
			fmt.Print(label)
		} else {
			fmt.Printf("%d", v)
		}
	case uint64:
		if label, ok := labels[int64(v)]; ok {
			fmt.Print(label)
		} else {
			fmt.Printf("%d", v)
		}
	default:
		printEDN(k, 0)
	}
}

func printVDP(v interface{}) {
	vdp, ok := v.(map[interface{}]interface{})
	if !ok {
		printEDN(v, 3)
		return
	}

	fmt.Println("{")
	for k, val := range vdp {
		fmt.Print("     ")
		printVDPLabel(k)
		fmt.Print(" ")
		printProof(k, val)
		fmt.Println()
	}
	fmt.Print("    }")
}

func printVDPLabel(k interface{}) {
	labels := map[int64]string{
		-1: ":inclusion-proof",
		-2: ":consistency-proof",
	}

	switch v := k.(type) {
	case int64:
		if label, ok := labels[v]; ok {
			fmt.Print(label)
		} else {
			fmt.Printf("%d", v)
		}
	case uint64:
		// Handle negative numbers encoded as uint64
		if v > (1<<63) {
			signed := int64(v)
			if label, ok := labels[signed]; ok {
				fmt.Print(label)
			} else {
				fmt.Printf("%d", signed)
			}
		} else {
			fmt.Printf("%d", v)
		}
	default:
		printEDN(k, 0)
	}
}

func printProof(k interface{}, v interface{}) {
	proofBytes, ok := v.([]byte)
	if !ok {
		printEDN(v, 4)
		return
	}

	// Decode the proof
	var proof interface{}
	if err := cbor.Unmarshal(proofBytes, &proof); err != nil {
		// If can't decode, show as hex
		printEDN(v, 4)
		return
	}

	// Check if it's inclusion or consistency proof
	proofArray, ok := proof.([]interface{})
	if !ok || len(proofArray) != 3 {
		printEDN(proof, 4)
		return
	}

	kInt := int64(0)
	switch kv := k.(type) {
	case int64:
		kInt = kv
	case uint64:
		if kv > (1 << 63) {
			kInt = int64(kv)
		} else {
			kInt = int64(kv)
		}
	}

	if kInt == -1 {
		// Inclusion proof: [tree-size, leaf-index, audit-path]
		fmt.Println("{")
		fmt.Print("      :tree-size ")
		printEDN(proofArray[0], 5)
		fmt.Println()
		fmt.Print("      :leaf-index ")
		printEDN(proofArray[1], 5)
		fmt.Println()
		fmt.Print("      :audit-path ")
		printHashArray(proofArray[2])
		fmt.Print("\n     }")
	} else if kInt == -2 {
		// Consistency proof: [old-size, new-size, proof-hashes]
		fmt.Println("{")
		fmt.Print("      :old-size ")
		printEDN(proofArray[0], 5)
		fmt.Println()
		fmt.Print("      :new-size ")
		printEDN(proofArray[1], 5)
		fmt.Println()
		fmt.Print("      :proof-hashes ")
		printHashArray(proofArray[2])
		fmt.Print("\n     }")
	} else {
		printEDN(proof, 4)
	}
}

func printHashArray(v interface{}) {
	hashes, ok := v.([]interface{})
	if !ok {
		printEDN(v, 5)
		return
	}

	fmt.Println("[")
	for _, hash := range hashes {
		fmt.Print("       ")
		printEDN(hash, 6)
		fmt.Println()
	}
	fmt.Print("      ]")
}

func printEDN(v interface{}, indent int) {
	switch val := v.(type) {
	case []interface{}:
		fmt.Print("[")
		for i, item := range val {
			if i > 0 {
				fmt.Print(" ")
			}
			printEDN(item, indent+1)
		}
		fmt.Print("]")

	case map[interface{}]interface{}:
		fmt.Print("{")
		first := true
		for k, v := range val {
			if !first {
				fmt.Print(" ")
			}
			first = false
			printEDN(k, indent+1)
			fmt.Print(" ")
			printEDN(v, indent+1)
		}
		fmt.Print("}")

	case []byte:
		// Print as hex string
		fmt.Printf("#h\"%s\"", hex.EncodeToString(val))

	case string:
		fmt.Printf("\"%s\"", val)

	case int64:
		fmt.Printf("%d", val)

	case uint64:
		fmt.Printf("%d", val)

	case int:
		fmt.Printf("%d", val)

	case uint:
		fmt.Printf("%d", val)

	case float64:
		fmt.Printf("%f", val)

	case bool:
		if val {
			fmt.Print("true")
		} else {
			fmt.Print("false")
		}

	case nil:
		fmt.Print("nil")

	default:
		fmt.Printf("#unknown<%T>", val)
	}
}
