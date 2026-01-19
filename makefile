default: build
CGO_ENABLED=1

execName = giftcode-get.exe

build:
	make build-linux
	make build-mac-arm64
	make build-windows

build-linux: GOOS = linux
build-linux: GOARCH = amd64
build-linux: build-base
	@echo "构建完成，输出文件在 target/type/$(GOOS)/$(GOARCH)/$(execName)"

build-mac-arm64: GOOS = darwin
build-mac-arm64: GOARCH = arm64
build-mac-arm64: build-base
	@echo "构建完成，输出文件在 target/type/$(GOOS)/$(GOARCH)/$(execName)"

build-windows: GOOS = windows
build-windows: GOARCH = amd64
build-windows: build-base
	@echo "构建完成，输出文件在 target/type/$(GOOS)/$(GOARCH)/$(execName)"


build-base:
	@if [ -z "$(GOOS)" ]; then echo "参数GOOS未设置"; exit 1; fi
	@if [ -z "$(GOARCH)" ]; then echo "参数GOARCH未设置"; exit 1; fi
	@echo "Building for $(GOOS) ${GOARCH}..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "-s -w" -o target/server/$(GOOS)/$(GOARCH)/$(execName) cmd/server/main.go
	chmod a+x target/server/$(GOOS)/$(GOARCH)/$(execName)
