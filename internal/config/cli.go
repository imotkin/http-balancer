package config

import (
	"flag"
	"time"
)

var (
	flagPath           = flag.String("config", "config.json", "Path for a config file")
	flagPort           = flag.Uint("port", 8080, "Port for HTTP server")
	flagHealthInterval = flag.Duration("health-interval", time.Second*5, "Health interval for endpoints")
	flagRefillInterval = flag.Duration("refill-interval", time.Millisecond*100, "Health interval for endpoints")
	flagEndpoints      = flag.String("endpoints", "[]", "List of endpoints in JSON format")
	flagLoggingLevel   = flag.String("logging", "info", "Logging level for balancer. Available options: debug, info, warn, error")
	flagStrategy       = flag.String("strategy", "round-robin", "Balancer strategy")
	flagMode           = flag.String("mode", "local", "Balancer mode")
	flagMigrationsPath = flag.String("migrations-path", "migrations", "Path for migrations")
	flagFilePath       = flag.String("file-path", "clients.sqlite", "Path for SQLite file")
)
