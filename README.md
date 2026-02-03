# PHP Tuner

A cross-platform CLI tool that analyzes your system and calculates optimal **FrankenPHP** or **PHP-FPM** configuration.

## Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/muuvmuuv/php-tuner/main/install.sh | sh
```

Or download a binary from the [releases page](https://github.com/muuvmuuv/php-tuner/releases).

## Features

- **Auto-detection** - Detects CPU cores, memory, and running PHP process memory usage
- **Cross-platform** - Works on Linux (amd64/arm64) and macOS (Intel/Apple Silicon)
- **FrankenPHP support** - Optimized `num_threads` and worker configuration (default)
- **PHP-FPM support** - All PM modes: `static`, `dynamic`, and `ondemand`
- **Traffic profiles** - Optimized recommendations for low, medium, and high traffic sites
- **Smart defaults** - Automatically reserves memory for OS and other services
- **Direct apply** - Apply changes directly to PHP-FPM config with `--apply`
- **Config export** - Pipe configuration directly to a file with `--config-only`

## Usage

```bash
php-tuner <command> [options]
```

### Commands

| Command | Alias | Description |
|---------|-------|-------------|
| `frankenphp` | `f` | Optimize FrankenPHP configuration (default) |
| `php-fpm` | `fpm` | Optimize PHP-FPM configuration |
| `help` | | Show help message |
| `version` | | Show version |

### FrankenPHP (Default)

```bash
# Auto-detect everything (runs FrankenPHP by default)
php-tuner

# Explicit FrankenPHP
php-tuner frankenphp

# Using shorthand
php-tuner f

# High-traffic production
php-tuner f --traffic high

# Export config to file
php-tuner f --config-only > Caddyfile.snippet
```

### PHP-FPM

```bash
# Auto-detect everything
php-tuner php-fpm

# Using shorthand
php-tuner fpm

# High-traffic production server
php-tuner fpm --traffic high --pm static

# Apply configuration directly
php-tuner fpm --apply --restart --yes

# Export config to file
php-tuner fpm --config-only > /tmp/php-fpm.conf
```

## Example Output

### FrankenPHP

```
FrankenPHP Optimizer
────────────────────────────────────────

System Information

  Platform             linux
  CPU Cores            4
  Total Memory         8192 MB

Calculation

  Reserved Memory      1076 MB (for OS/Caddy)
  Available for PHP    7116 MB
  Thread Memory        30.0 MB
  Formula              7116 MB / 30.0 MB = 8 threads

Recommended Configuration

{
    frankenphp {
        num_threads 8
        max_threads 16
        max_wait_time 10s
        worker {
            file /path/to/your/public/index.php
            num 8
        }
    }
}
```

### PHP-FPM

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

### FrankenPHP Options

```
php-tuner frankenphp [options]
php-tuner f [options]
```

| Flag | Description |
|------|-------------|
| `-h, --help` | Show help message |
| `-c, --config-only` | Output only the configuration |
| `--no-color` | Disable colored output |
| `--traffic <level>` | Traffic profile: `low`, `medium`, `high` |
| `--reserved <MB>` | Override reserved memory for OS/Caddy |
| `--thread-mem <MB>` | Override estimated thread memory (default: 30MB) |
| `--worker=false` | Disable worker mode |

### PHP-FPM Options

```
php-tuner php-fpm [options]
php-tuner fpm [options]
```

| Flag | Description |
|------|-------------|
| `-h, --help` | Show help message |
| `-c, --config-only` | Output only the configuration |
| `--no-color` | Disable colored output |
| `--pm <type>` | Force PM type: `static`, `dynamic`, `ondemand` |
| `--traffic <level>` | Traffic profile: `low`, `medium`, `high` |
| `--reserved <MB>` | Override reserved memory for OS/services |
| `--process-mem <MB>` | Override detected PHP process memory |
| `--apply` | Apply configuration directly to config file |
| `--config <path>` | Path to PHP-FPM pool config |
| `--restart` | Restart PHP-FPM service after applying |
| `-y, --yes` | Skip confirmation prompts |

## Traffic Profiles

| Profile | FrankenPHP | PHP-FPM |
|---------|------------|---------|
| `low` | Fewer threads, no wait timeout | `ondemand` PM, longer idle timeout |
| `medium` | Balanced threads | `dynamic` PM |
| `high` | More threads, strict timeouts | `static` PM |

## How It Works

### FrankenPHP

FrankenPHP uses threads instead of processes, which share memory and are more efficient:

```
num_threads = min(CPU Cores × 2, Available Memory / Thread Memory)
max_threads = CPU Cores × 4
worker_num = num_threads (for worker mode)
```

Reserved memory: `256MB + 10% of total RAM` (capped at 2GB)

### PHP-FPM

Uses the formula from [Tideways' PHP-FPM tuning guide](https://tideways.com/profiler/blog/an-introduction-to-php-fpm-tuning):

```
max_children = (Total RAM - Reserved Memory) / Average PHP Process Memory
start_servers = CPU Cores × 4
min_spare_servers = CPU Cores × 2  
max_spare_servers = CPU Cores × 4
```

Reserved memory: `512MB + 15% of total RAM` (capped at 4GB)

## Building from Source

Requires Go 1.21+ and [just](https://github.com/casey/just)

```bash
git clone https://github.com/muuvmuuv/php-tuner.git
cd php-tuner

just build        # Build for current platform
just build-all    # Build for all platforms
just install      # Install to /usr/local/bin
just run f        # Build and run FrankenPHP
just run fpm      # Build and run PHP-FPM
```

## Applying Configuration

### FrankenPHP

Add the output to your Caddyfile:

```bash
# Generate and append to Caddyfile
php-tuner f --config-only >> /etc/frankenphp/Caddyfile

# Reload
frankenphp reload
```

### PHP-FPM

Use `--apply` for automatic configuration:

```bash
# Apply with confirmation
php-tuner fpm --apply

# Apply and restart
php-tuner fpm --apply --restart

# Skip confirmation (CI/automation)
php-tuner fpm --apply --restart --yes
```

Or manually copy to your pool config:

```bash
php-tuner fpm --config-only >> /etc/php/8.x/fpm/pool.d/www.conf
sudo systemctl restart php-fpm
```

## License

MIT

## References

- [FrankenPHP Configuration](https://frankenphp.dev/docs/config/) - FrankenPHP Docs
- [An Introduction to PHP-FPM Tuning](https://tideways.com/profiler/blog/an-introduction-to-php-fpm-tuning) - Tideways
- [PHP-FPM Configuration](https://www.php.net/manual/en/install.fpm.configuration.php) - PHP Manual
