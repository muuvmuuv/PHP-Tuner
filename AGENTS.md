# Agent Instructions

## Git Operations

**NEVER** perform these operations without explicit user approval:

- `git commit`
- `git push`
- `git tag`
- `just release`

Approval is required **each time** - a single approval does not carry over to subsequent operations. Always wait for explicit confirmation before executing any of these commands.

## Project Context

PHP Tuner is a CLI tool that analyzes system resources and calculates optimal configuration for:

- **FrankenPHP** (default) - modern PHP application server
- **PHP-FPM** - traditional PHP FastCGI process manager

## Tech Stack

- Language: Go
- Build: `just` (justfile)
- Release: GoReleaser + GitHub Actions
- Platforms: linux/amd64, linux/arm64

## Key Files

```
cmd/php-tuner/
├── main.go         # Entry point, command routing
├── frankenphp.go   # FrankenPHP subcommand
└── phpfpm.go       # PHP-FPM subcommand

internal/
├── calculator/     # Config calculation logic
├── output/         # Terminal output formatting
├── php/            # PHP process detection
└── system/         # System info detection
```

## Commands

```bash
just build          # Build binary
just run <args>     # Build and run
just test           # Run tests
just release <ver>  # Create release (requires approval)
```
