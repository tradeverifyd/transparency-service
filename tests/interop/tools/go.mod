module github.com/tradeverifyd/scitt/tests/interop/tools

go 1.24.0

toolchain go1.24.8

require github.com/tradeverifyd/transparency-service/scitt-golang v0.0.0

require (
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/veraison/go-cose v1.3.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
)

replace github.com/tradeverifyd/transparency-service/scitt-golang => ../../../scitt-golang
