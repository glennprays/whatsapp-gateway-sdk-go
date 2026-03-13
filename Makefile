.PHONY: generate build test clean

# Generate types from OpenAPI spec
generate:
	@which oapi-codegen > /dev/null || go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	oapi-codegen -generate types -package generated ../whatsapp-gateway/docs/openapi.yaml > generated/types.gen.go

# Build the SDK
build:
	go build ./...

# Run tests
test:
	go test -v ./...

# Clean generated files
clean:
	rm -f generated/types.gen.go

# All-in-one: generate and test
all: generate test
