module bitcoin-sprint

go 1.23.0

// Build & Link Instructions:
// 1. Build Rust secure module:
//    cd ../../secure/rust && cargo build --release
// 2. Build Go application:
//    go build -o bitcoin-sprint.exe

require (
	github.com/PayRpc/Bitcoin-Sprint v0.0.0-00010101000000-000000000000
	go.uber.org/zap v1.27.0
	golang.org/x/time v0.12.0
)

require go.uber.org/multierr v1.10.0 // indirect

replace github.com/PayRpc/Bitcoin-Sprint => ../..
