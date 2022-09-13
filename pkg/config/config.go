package config

import (
	"flag"
	"os"
)

type Config struct {
	RunAddress           string
	DatabaseURI          string
	AccrualSystemAddress string
	LogLevel             string
	SecretKey            string
}

func Parse() *Config {
	cfg := Config{
		// Defaults
		RunAddress:           "localhost:8080",
		AccrualSystemAddress: "localhost:8888",
		SecretKey:            "secret",
		LogLevel:             "debug",
	}
	cfg.updateFromFlags()
	cfg.updateFromEnv()
	return &cfg
}

func (cfg *Config) updateFromFlags() {
	flagRunAddress := flag.String("a", cfg.RunAddress, "Server address.")
	flagDatabaseURI := flag.String("d", cfg.DatabaseURI, "Postgres DSN.")
	flagAccrualAddress := flag.String("r", cfg.AccrualSystemAddress, "Accrual System address.")

	flag.Parse()

	cfg.RunAddress = *flagRunAddress
	cfg.DatabaseURI = *flagDatabaseURI
	cfg.AccrualSystemAddress = *flagAccrualAddress
}

func (cfg *Config) updateFromEnv() {
	if addr, ok := os.LookupEnv("RUN_ADDRESS"); ok {
		cfg.RunAddress = addr
	}
	if db, ok := os.LookupEnv("DATABASE_URI"); ok {
		cfg.DatabaseURI = db
	}
	if addr, ok := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS"); ok {
		cfg.AccrualSystemAddress = addr
	}
	if secret, ok := os.LookupEnv("SECRET_KEY"); ok {
		cfg.SecretKey = secret
	}
	if lvl, ok := os.LookupEnv("LOG_LEVEL"); ok {
		cfg.LogLevel = lvl
	}
}
