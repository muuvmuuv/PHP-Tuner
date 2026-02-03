# PHP Tuner

binary := "php-tuner"
version := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`

default: build

build:
    go build -ldflags "-s -w -X main.version={{version}}" -o dist/{{binary}} ./cmd/php-tuner

build-all: clean
    #!/usr/bin/env sh
    for target in linux/amd64 linux/arm64; do
        os="${target%/*}"
        arch="${target#*/}"
        echo "Building ${os}/${arch}..."
        GOOS=$os GOARCH=$arch go build -ldflags "-s -w -X main.version={{version}}" \
            -o "dist/{{binary}}-${os}-${arch}" ./cmd/php-tuner
    done

clean:
    rm -rf dist

test:
    go test -v ./...

run *args: build
    ./dist/{{binary}} {{args}}

install: build
    sudo cp dist/{{binary}} /usr/local/bin/{{binary}}

fmt:
    go fmt ./...

lint:
    go vet ./...

# Create release: just release 0.1.0
release tag:
    git tag -s "v{{tag}}" -m "Release v{{tag}}"
    git push origin "v{{tag}}"

release-dry:
    goreleaser release --snapshot --clean
