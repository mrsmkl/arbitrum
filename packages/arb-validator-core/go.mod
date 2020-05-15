module github.com/offchainlabs/arbitrum/packages/arb-validator-core

go 1.12

require (
	github.com/OneOfOne/xxhash v1.2.5 // indirect
	github.com/ethereum/go-ethereum v1.9.14
	github.com/golang/protobuf v1.4.1
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/mux v1.7.4
	github.com/offchainlabs/arbitrum/packages/arb-util v0.5.0
	github.com/pkg/errors v0.9.1
	github.com/robertkrimen/otto v0.0.0-20170205013659-6a77b7cbc37d // indirect
	github.com/spaolacci/murmur3 v1.0.1-0.20190317074736-539464a789e9 // indirect
	google.golang.org/grpc v1.29.1
	google.golang.org/protobuf v1.22.0
	gopkg.in/sourcemap.v1 v1.0.5 // indirect
)

replace github.com/offchainlabs/arbitrum/packages/arb-util => ../arb-util
