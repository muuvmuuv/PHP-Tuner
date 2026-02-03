# PHP-FPM Optimizer

binary_name := "php-fpm-optimizer"
build_dir := "dist"
version := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`
ldflags := "-s -w -X main.version=" + version

# Default recipe
default: build

# Build for current platform
build:
    @echo "Building {{binary_name}}..."
    @mkdir -p {{build_dir}}
    go build -ldflags "{{ldflags}}" -o {{build_dir}}/{{binary_name}} ./cmd/php-fpm-optimizer

# Build for all supported platforms
build-all: clean
    @echo "Building for all platforms..."
    @mkdir -p {{build_dir}}
    
    @echo "  → linux/amd64"
    GOOS=linux GOARCH=amd64 go build -ldflags "{{ldflags}}" -o {{build_dir}}/{{binary_name}}-linux-amd64 ./cmd/php-fpm-optimizer
    
    @echo "  → linux/arm64"
    GOOS=linux GOARCH=arm64 go build -ldflags "{{ldflags}}" -o {{build_dir}}/{{binary_name}}-linux-arm64 ./cmd/php-fpm-optimizer
    
    @echo "  → darwin/amd64"
    GOOS=darwin GOARCH=amd64 go build -ldflags "{{ldflags}}" -o {{build_dir}}/{{binary_name}}-darwin-amd64 ./cmd/php-fpm-optimizer
    
    @echo "  → darwin/arm64"
    GOOS=darwin GOARCH=arm64 go build -ldflags "{{ldflags}}" -o {{build_dir}}/{{binary_name}}-darwin-arm64 ./cmd/php-fpm-optimizer
    
    @echo "Done!"
    @ls -lh {{build_dir}}/

# Remove build artifacts
clean:
    @echo "Cleaning..."
    rm -rf {{build_dir}}

# Run tests
test:
    go test -v ./...

# Build and run
run *args: build
    ./{{build_dir}}/{{binary_name}} {{args}}

# Install to /usr/local/bin
install: build
    @echo "Installing to /usr/local/bin/{{binary_name}}..."
    sudo cp {{build_dir}}/{{binary_name}} /usr/local/bin/{{binary_name}}
    @echo "Done!"

# Format code
fmt:
    go fmt ./...

# Run linter
lint:
    golangci-lint run

# Show version that would be embedded
version:
    @echo {{version}}
