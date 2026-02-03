# PHP Tuner

[![Release](https://github.com/muuvmuuv/PHP-Tuner/actions/workflows/release.yaml/badge.svg)](https://github.com/muuvmuuv/PHP-Tuner/actions/workflows/release.yaml)

A CLI tool that analyzes your system and calculates optimal **FrankenPHP** or **PHP-FPM** configuration.

## Quick Start

Run directly without installing:

```bash
# Linux (amd64)
curl -fsSL https://github.com/muuvmuuv/php-tuner/releases/latest/download/php-tuner-linux-amd64.tar.gz | tar xz && ./php-tuner

# Linux (arm64)
curl -fsSL https://github.com/muuvmuuv/php-tuner/releases/latest/download/php-tuner-linux-arm64.tar.gz | tar xz && ./php-tuner

# PHP-FPM instead of FrankenPHP
./php-tuner fpm
```

## Usage

```
php-tuner <command> [options]

Commands:
    frankenphp, f    FrankenPHP configuration (default)
    php-fpm, fpm     PHP-FPM configuration
    help             Show help
    version          Show version
```

### FrankenPHP (Default)

```bash
php-tuner                       # Auto-detect
php-tuner f --traffic high      # High-traffic profile
php-tuner f -c > config.txt     # Export config only
```

### PHP-FPM

```bash
php-tuner fpm                           # Auto-detect
php-tuner fpm --traffic high --pm static
php-tuner fpm -c > www.conf             # Export config only
```

## Options

### FrankenPHP

| Flag | Description |
|------|-------------|
| `-c, --config-only` | Output only configuration |
| `--no-color` | Disable colors |
| `--traffic <level>` | `low`, `medium`, `high` |
| `--reserved <MB>` | Reserved memory for OS/Caddy |
| `--thread-mem <MB>` | Override thread memory estimate |
| `--worker=false` | Disable worker mode |

### PHP-FPM

| Flag | Description |
|------|-------------|
| `-c, --config-only` | Output only configuration |
| `--no-color` | Disable colors |
| `--pm <type>` | `static`, `dynamic`, `ondemand` |
| `--traffic <level>` | `low`, `medium`, `high` |
| `--reserved <MB>` | Reserved memory for OS |
| `--process-mem <MB>` | Override process memory |

## Traffic Profiles

| Profile | FrankenPHP | PHP-FPM |
|---------|------------|---------|
| `low` | Fewer threads | `ondemand` PM |
| `medium` | Balanced | `dynamic` PM |
| `high` | More threads, strict timeouts | `static` PM |

## How It Works

### FrankenPHP

```
num_threads = min(CPU × 2, Available Memory / 30MB)
max_threads = CPU × 4
```

### PHP-FPM

Based on [Tideways' tuning guide](https://tideways.com/profiler/blog/an-introduction-to-php-fpm-tuning):

```
max_children = (RAM - Reserved) / Process Memory
start_servers = CPU × 4
min_spare_servers = CPU × 2
max_spare_servers = CPU × 4
```

## Building

Requires Go 1.21+ and [just](https://github.com/casey/just)

```bash
just build      # Build
just run f      # Build and run
just test       # Run tests
```

## License

MIT
