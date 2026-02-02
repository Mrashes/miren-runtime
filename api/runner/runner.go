package runner

//go:generate mkdir -p runner_v1alpha
//go:generate go run ../../pkg/entity/cmd/schemagen -input schema.yml -output runner_v1alpha/schema.gen.go -pkg runner_v1alpha
//go:generate go run ../../pkg/rpc/cmd/rpcgen -pkg runner_v1alpha -input rpc.yml -output runner_v1alpha/rpc.gen.go
