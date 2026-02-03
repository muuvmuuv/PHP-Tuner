# PHP-FPM Optimizer

A cross-platform CLI tool that analyzes your system and calculates optimal PHP-FPM process manager configuration.

## Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/php-fpm/optimizer/main/install.sh | sh
```

Or download a binary from the [releases page](https://github.com/php-fpm/optimizer/releases).

## Features

- **Auto-detection** - Detects CPU cores, memory, and running PHP-FPM process memory usage
- **Cross-platform** - Works on Linux (amd64/arm64) and macOS (Intel/Apple Silicon)
- **All PM modes** - Supports `static`, `dynamic`, and `ondemand` configurations
- **Traffic profiles** - Optimized recommendations for low, medium, and high traffic sites
- **Smart defaults** - Automatically reserves memory for OS and other services
- **Direct apply** - Apply changes directly to PHP-FPM config with `--apply`
- **Config export** - Pipe configuration directly to a file with `--config-only`

## Usage

```bash
# Auto-detect everything (recommended for most users)
php-fpm-optimizer

# High-traffic production server
php-fpm-optimizer --traffic high --pm static

# Low-memory VPS with light traffic  
php-fpm-optimizer --traffic low

# Export config directly to file
php-fpm-optimizer --config-only > /tmp/php-fpm-pm.conf

# Specify known process memory usage
php-fpm-optimizer --process-mem 64
```

## Example Output

```
PHP-FPM Process Manager Optimizer
────────────────────────────────────────

System Information

  Platform             linux
  CPU Cores            4
  Total Memory         8192 MB
  Available Memory     5120 MB
  Used Memory          3072 MB

PHP-FPM Processes

  Process Count        12
  Average Memory       62.4 MB
  Total Memory         748.8 MB

Calculation

  Reserved Memory      1740 MB (for OS/services)
  Available for PHP    6452 MB
  Process Memory       62.4 MB
  Formula              6452 MB / 62.4 MB = 103 workers

Recommended Configuration

pm = dynamic
pm.max_children = 103
pm.process_idle_timeout = 5s
pm.start_servers = 16
pm.min_spare_servers = 8
pm.max_spare_servers = 16
pm.max_requests = 500
```

## Options

| Flag | Description |
|------|-------------|
| `-h, --help` | Show help message |
| `-v, --version` | Show version |
| `-c, --config-only` | Output only the configuration (for piping) |
| `--no-color` | Disable colored output |
| `--pm <type>` | Force PM type: `static`, `dynamic`, `ondemand` |
| `--traffic <level>` | Traffic profile: `low`, `medium`, `high` |
| `--reserved <MB>` | Override reserved memory for OS/services |
| `--process-mem <MB>` | Override detected PHP process memory |
| `--apply` | Apply configuration directly to PHP-FPM config file |
| `--config <path>` | Path to PHP-FPM pool config (default: auto-detect) |
| `--restart` | Restart PHP-FPM service after applying |
| `-y, --yes` | Skip confirmation prompts |

## Process Manager Types

| Type | Best For | Behavior |
|------|----------|----------|
| `static` | High-traffic sites | Fixed number of workers always running |
| `dynamic` | Most use cases | Workers scale between min/max based on demand |
| `ondemand` | Low-traffic / shared hosting | Workers spawn on demand, killed when idle |

## Traffic Profiles

- **low** - Selects `ondemand` PM, longer idle timeout. Best for admin panels, staging sites.
- **medium** - Selects `dynamic` PM. Balanced for typical web applications.
- **high** - Selects `static` PM, shorter idle timeout. Best for production sites with consistent traffic.

## How It Works

The optimizer uses the formula from [Tideways' PHP-FPM tuning guide](https://tideways.com/profiler/blog/an-introduction-to-php-fpm-tuning):

```
max_children = (Total RAM - Reserved Memory) / Average PHP Process Memory
start_servers = CPU Cores × 4
min_spare_servers = CPU Cores × 2  
max_spare_servers = CPU Cores × 4
```

Reserved memory is auto-calculated as: `512MB + 15% of total RAM` (capped at 4GB).

## Building from Source

Requires Go 1.21+ and [just](https://github.com/casey/just)

```bash
# Clone the repository
git clone https://github.com/php-fpm/optimizer.git
cd optimizer

# Build for current platform
just build

# Build for all platforms
just build-all

# Install to /usr/local/bin
just install

# Build and run with arguments
just run --traffic high

# List all available commands
just --list
```

## Applying Configuration

### Automatic (Recommended)

Use `--apply` to directly update your PHP-FPM configuration:

```bash
# Apply with confirmation prompt
php-fpm-optimizer --apply

# Apply and restart PHP-FPM
php-fpm-optimizer --apply --restart

# Skip confirmation (for scripts/automation)
php-fpm-optimizer --apply --restart --yes

# Apply to a specific config file
php-fpm-optimizer --apply --config /etc/php/8.2/fpm/pool.d/www.conf
```

The tool will:
1. Auto-detect your PHP-FPM pool configuration
2. Create a backup (`.backup` file)
3. Update only the PM-related settings
4. Optionally restart PHP-FPM

### Manual

1. Copy the output to your PHP-FPM pool configuration:
   ```
   /etc/php/8.x/fpm/pool.d/www.conf
   ```

2. Restart PHP-FPM:
   ```bash
   sudo systemctl restart php-fpm
   # or
   sudo systemctl restart php8.3-fpm
   ```

## Advanced: Multiple Pools

For applications with distinct frontend/backend traffic patterns, consider separate pools:

```ini
; /etc/php/8.x/fpm/pool.d/frontend.conf
[frontend]
listen = /var/run/php-fpm-frontend.sock
pm = static
pm.max_children = 50

; /etc/php/8.x/fpm/pool.d/backend.conf  
[backend]
listen = /var/run/php-fpm-backend.sock
pm = ondemand
pm.max_children = 10
pm.process_idle_timeout = 10s
```

## License

MIT

## References

- [An Introduction to PHP-FPM Tuning](https://tideways.com/profiler/blog/an-introduction-to-php-fpm-tuning) - Tideways
- [PHP-FPM Configuration](https://www.php.net/manual/en/install.fpm.configuration.php) - PHP Manual
