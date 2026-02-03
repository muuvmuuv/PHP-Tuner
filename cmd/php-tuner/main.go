package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		// Default to frankenphp
		os.Args = append(os.Args, "frankenphp")
	}

	cmd := os.Args[1]

	switch cmd {
	case "frankenphp", "f":
		runFrankenPHP(os.Args[2:])
	case "php-fpm", "fpm":
		runPHPFPM(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Printf("php-tuner %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		// Check if it looks like a flag (for backwards compat or misuse)
		if len(cmd) > 0 && cmd[0] == '-' {
			fmt.Fprintf(os.Stderr, "Unknown flag: %s\n\n", cmd)
		} else {
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		}
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`PHP Tuner - Optimize your PHP runtime configuration

USAGE:
    php-tuner <command> [options]

COMMANDS:
    frankenphp, f    Optimize FrankenPHP configuration (default)
    php-fpm, fpm     Optimize PHP-FPM configuration
    help             Show this help message
    version          Show version

EXAMPLES:
    php-tuner                           # FrankenPHP (default)
    php-tuner frankenphp                # FrankenPHP explicitly  
    php-tuner f --traffic high          # FrankenPHP shorthand
    php-tuner php-fpm                   # PHP-FPM
    php-tuner fpm --apply --restart     # PHP-FPM with auto-apply

Run 'php-tuner <command> --help' for more information on a command.

QUICK INSTALL:
    curl -fsSL https://raw.githubusercontent.com/muuvmuuv/php-tuner/main/install.sh | sh

For more information, visit: https://github.com/muuvmuuv/php-tuner`)
}
