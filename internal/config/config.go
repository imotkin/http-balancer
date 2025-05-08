package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"time"

	"github.com/joho/godotenv"
)

var (
	logLevels = map[string]slog.Level{
		"none":  -1,
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}

	strategies = []string{
		"round-robin",
		"least-connections",
		"random",
	}

	modes = []string{
		"local",
		"remote",
	}
)

// Дополнительный тип данных для работы с time.Duration
// Требуется для правильной десериализации значений интервалов
// (HealthInterval, RefillInterval) из JSON
type Duration struct {
	time.Duration
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string

	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	d.Duration, err = time.ParseDuration(s)
	if err != nil {
		return err
	}

	return nil
}

// Возвращает стандартную конфигурацию
func Default() *Config {
	return &Config{
		Port:           8080,
		Strategy:       "round-robin",
		HealthInterval: Duration{5 * time.Second},
		RefillInterval: Duration{100 * time.Millisecond},
		LoggingLevel:   "error",
		Defaults: Defaults{
			Capacity: 10,
			Rate:     1,
		},
		Mode: "local",
	}
}

type Config struct {
	// Уровень логирования событий для балансировщика
	LoggingLevel string `json:"logging"`

	// Порт для работы сервера балансировщика
	Port uint `json:"port"`

	// Список URL-адресов для серверов балансировщика
	Endpoints []string `json:"endpoints"`

	// Выбранная стратегия для работы балансировщика (round-robin, least-connections, random)
	Strategy string `json:"strategy"`

	// Интервал для проверки (ping) текущего состояния всех серверов балансировщика
	HealthInterval Duration `json:"healthInterval"`

	// Интервал для добавления новых токенов в Token Bucket
	RefillInterval Duration `json:"refillInterval"`

	// Стандартные значения параметров для клиента в Token Bucket
	Defaults Defaults `json:"defaults"`

	// Режим работы хранилища для клиентов (локальный - local, удалённый - remote)
	Mode string `json:"mode"`

	// Путь для директории с файлами для миграции
	MigrationsPath string `json:"migrationsPath"`

	// Путь для локального файла SQLite
	FilePath string `json:"filePath"`

	// Данные для подключения к PostgreSQL
	Database struct {
		Host     string
		Name     string
		Password string
		Port     string
		User     string
	}
}

// Стандартные значения для ёмкости и скорости пополнения Token Bucket
type Defaults struct {
	Capacity uint `json:"capacity"`
	Rate     uint `json:"rate"`
}

// Функция для валидации данных конфигурации
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("empty config is passed")
	}

	if _, ok := logLevels[c.LoggingLevel]; !ok {
		return errors.New("invalid balancer logging level")
	}

	if c.Port == 0 {
		return errors.New("null server port")
	}

	if len(c.Endpoints) == 0 {
		return errors.New("list of endpoints is empty")
	}

	if !slices.Contains(strategies, c.Strategy) {
		return errors.New("invalid balancer strategy")
	}

	if c.HealthInterval.Duration == 0 {
		return errors.New("null health interval")
	}

	if c.RefillInterval.Duration == 0 {
		return errors.New("null refill interval")
	}

	if c.Defaults.Capacity == 0 {
		return errors.New("null default capacity")
	}

	if c.Defaults.Rate == 0 {
		return errors.New("null default rate")
	}

	if !slices.Contains(modes, c.Mode) {
		return errors.New("invalid balancer mode")
	}

	if c.MigrationsPath == "" {
		return errors.New("empty migrations path")
	}

	if c.Mode == "local" && c.FilePath == "" {
		return errors.New("empty file path in local mode")
	}

	return nil
}

// Возвращает уровень логгера на основе данных из текущей конфигурации
func (c *Config) LogLevel() slog.Level {
	return logLevels[c.LoggingLevel]
}

// Возвращает в виде строки данные подключения к PostgreSQL
func (c *Config) DatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Name,
	)
}

func Load() (*Config, error) {
	flag.Parse()

	var hasPort, hasEndpoints bool

	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "port":
			hasPort = true
		case "endpoints":
			hasEndpoints = true
		default:
			return
		}
	})

	if hasPort && hasEndpoints {
		var endpoints []string

		err := json.Unmarshal([]byte(*flagEndpoints), &endpoints)
		if err != nil {
			return nil, fmt.Errorf("decode endpoints JSON list: %w", err)
		}

		if _, ok := logLevels[*flagLoggingLevel]; !ok {
			return nil, fmt.Errorf("invalid logging level: %v", *flagLoggingLevel)
		}

		return &Config{
			Port:           *flagPort,
			Endpoints:      endpoints,
			HealthInterval: Duration{*flagHealthInterval},
			RefillInterval: Duration{*flagRefillInterval},
			LoggingLevel:   *flagLoggingLevel,
			Mode:           *flagMode,
			Strategy:       *flagStrategy,
			MigrationsPath: *flagMigrationsPath,
			FilePath:       *flagFilePath,
		}, nil
	}

	var cfg Config

	file, err := os.Open(*flagPath)
	if err != nil {
		return nil, err
	}

	err = json.NewDecoder(file).Decode(&cfg)
	if err != nil {
		return nil, err
	}

	// Если удалённый режим работы, то необходимо
	// добавить данные для подключения к PostgreSQL
	if cfg.Mode == "remote" {
		err := godotenv.Load()
		if err != nil {
			return nil, fmt.Errorf("load .env file: %w", err)
		}

		cfg.Database.Host = os.Getenv("POSTGRES_HOST")
		cfg.Database.Name = os.Getenv("POSTGRES_DB")
		cfg.Database.Password = os.Getenv("POSTGRES_PASSWORD")
		cfg.Database.Port = os.Getenv("POSTGRES_PORT")
		cfg.Database.User = os.Getenv("POSTGRES_USER")
	}

	return &cfg, err
}
