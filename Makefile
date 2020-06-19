.PHONY: dsplinz android ios dsplinz-cross swarm evm all test clean
.PHONY: dsplinz-linux dsplinz-linux-386 dsplinz-linux-amd64 dsplinz-linux-mips64 dsplinz-linux-mips64le
.PHONY: dsplinz-linux-arm dsplinz-linux-arm-5 dsplinz-linux-arm-6 dsplinz-linux-arm-7 dsplinz-linux-arm64
.PHONY: dsplinz-darwin dsplinz-darwin-386 dsplinz-darwin-amd64
.PHONY: dsplinz-windows dsplinz-windows-386 dsplinz-windows-amd64

GOBIN = $(shell pwd)/build/bin
GO ?= latest

dsplinz:
	build/env.sh go run build/ci.go install ./cmd/dsplinz
	@echo "Done building."
	@echo "Run \"$(GOBIN)/dsplinz\" to launch dsplinz."

swarm:
	build/env.sh go run build/ci.go install ./cmd/swarm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swarm\" to launch swarm."

all:
	build/env.sh go run build/ci.go install

test: all
	build/env.sh go run build/ci.go test

lint: ## Run linters.
	build/env.sh go run build/ci.go lint

clean:
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

# Cross Compilation Targets (xgo)

dsplinz-cross: dsplinz-linux dsplinz-darwin dsplinz-windows dsplinz-android dsplinz-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/dsplinz-*

dsplinz-linux: dsplinz-linux-386 dsplinz-linux-amd64 dsplinz-linux-arm dsplinz-linux-mips64 dsplinz-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/dsplinz-linux-*

dsplinz-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/dsplinz
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/dsplinz-linux-* | grep 386

dsplinz-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/dsplinz
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/dsplinz-linux-* | grep amd64

dsplinz-linux-arm: dsplinz-linux-arm-5 dsplinz-linux-arm-6 dsplinz-linux-arm-7 dsplinz-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/dsplinz-linux-* | grep arm

dsplinz-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/dsplinz
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/dsplinz-linux-* | grep arm-5

dsplinz-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/dsplinz
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/dsplinz-linux-* | grep arm-6

dsplinz-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/dsplinz
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/dsplinz-linux-* | grep arm-7

dsplinz-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/dsplinz
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/dsplinz-linux-* | grep arm64

dsplinz-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/dsplinz
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/dsplinz-linux-* | grep mips

dsplinz-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/dsplinz
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/dsplinz-linux-* | grep mipsle

dsplinz-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/dsplinz
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/dsplinz-linux-* | grep mips64

dsplinz-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/dsplinz
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/dsplinz-linux-* | grep mips64le

dsplinz-darwin: dsplinz-darwin-386 dsplinz-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/dsplinz-darwin-*

dsplinz-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/dsplinz
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/dsplinz-darwin-* | grep 386

dsplinz-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/dsplinz
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/dsplinz-darwin-* | grep amd64

dsplinz-windows: dsplinz-windows-386 dsplinz-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/dsplinz-windows-*

dsplinz-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/dsplinz
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/dsplinz-windows-* | grep 386

dsplinz-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/dsplinz
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/dsplinz-windows-* | grep amd64
