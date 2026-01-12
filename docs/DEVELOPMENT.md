# Development Guide

This document provides instructions for developing and contributing to TxHammer.

## Prerequisites

- Go 1.21 or higher
- Make (optional, for convenience commands)
- golangci-lint (for linting)

## Building

```bash
# Build using make
make build

# Or build directly
go build -o build/txhammer ./cmd
```

The binary will be created at `build/txhammer`.

## Testing

```bash
# Run all tests
make test

# Or run directly
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test ./internal/wallet/...
```

## Code Quality

```bash
# Run linter
make lint

# Format code
make fmt

# Run all checks (lint + test)
make check
```

## Project Cleanup

```bash
# Remove build artifacts
make clean
```

## Module Structure

Each internal package follows a consistent structure:

```
internal/<package>/
├── types.go      # Type definitions and interfaces
├── <package>.go  # Main implementation
└── <package>_test.go # Unit tests
```

## Adding a New Transaction Mode

1. Create a new builder in `internal/txbuilder/`:
   ```go
   type MyModeBuilder struct {
       // fields
   }

   func (b *MyModeBuilder) Build(ctx context.Context, opts BuildOptions) ([]*types.Transaction, error) {
       // implementation
   }
   ```

2. Register the builder in `internal/txbuilder/factory.go`:
   ```go
   case ModeMyMode:
       return NewMyModeBuilder(cfg), nil
   ```

3. Add the mode constant in `internal/config/config.go`:
   ```go
   const (
       ModeMyMode Mode = "MY_MODE"
   )
   ```

4. Update CLI flags in `cmd/main.go` if needed

## Testing Guidelines

- Use table-driven tests for comprehensive coverage
- Mock external dependencies (RPC client, etc.)
- Test error conditions and edge cases
- Aim for >80% code coverage

Example test structure:
```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   InputType
        want    OutputType
        wantErr bool
    }{
        {
            name:  "valid input",
            input: validInput,
            want:  expectedOutput,
        },
        {
            name:    "invalid input",
            input:   invalidInput,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("MyFunction() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("MyFunction() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Debugging

Enable verbose logging with the `--verbose` flag:

```bash
./build/txhammer --verbose ...
```

For development, you can also use Go's race detector:

```bash
go build -race -o build/txhammer ./cmd
```
