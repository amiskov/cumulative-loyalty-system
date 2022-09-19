package config

import (
	"flag"
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	RunAddress            string
	DatabaseURI           string
	AccrualSystemAddress  string        // full address, like `http//localhost:8888`
	AccrualPollingLimit   int           // max attempts to get order info from accrual system
	AccrualPollingTimeout time.Duration // timeout between attempts to get order info from accrual system
	LogLevel              string
	SecretKey             string
}

func Parse() *Config {
	cfg := Config{
		// Defaults
		RunAddress:            "localhost:8080",
		AccrualSystemAddress:  "http://localhost:8888",
		AccrualPollingLimit:   100,
		AccrualPollingTimeout: 1 * time.Second,
		SecretKey:             "secret",
		LogLevel:              "debug",
	}
	cfg.updateFromFlags()
	cfg.updateFromEnv()
	return &cfg
}

func (cfg *Config) updateFromFlags() {
	flagRunAddress := flag.String("a", cfg.RunAddress, "Server address.")
	flagDatabaseURI := flag.String("d", cfg.DatabaseURI, "Postgres DSN.")
	flagAccrualAddress := flag.String("r", cfg.AccrualSystemAddress, "Accrual System address.")
	flagAccrualPollingTimeout := flag.Duration("i", cfg.AccrualPollingTimeout,
		"Pause between attempts to get order accrual.")
	flagAccrualPollingLimit := flag.Int("l", cfg.AccrualPollingLimit,
		"Max attempts to get order accrual from the accrual system.")

	flag.Parse()

	cfg.RunAddress = *flagRunAddress
	cfg.DatabaseURI = *flagDatabaseURI
	cfg.AccrualSystemAddress = *flagAccrualAddress
	cfg.AccrualPollingLimit = *flagAccrualPollingLimit
	cfg.AccrualPollingTimeout = *flagAccrualPollingTimeout
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
	if lim, ok := os.LookupEnv("ACCRUAL_POLLING_LIMIT"); ok {
		limit, err := strconv.Atoi(lim)
		if err != nil {
			log.Fatal("bad accrual polling limit value, must be int (times)")
		}
		cfg.AccrualPollingLimit = limit
	}
	if timeout, ok := os.LookupEnv("ACCRUAL_POLLING_TIMEOUT"); ok {
		t, err := strconv.Atoi(timeout)
		if err != nil {
			log.Fatal("bad accrual polling timeout value, must be int (seconds)")
		}
		cfg.AccrualPollingTimeout = time.Duration(t) * time.Second
	}
	if secret, ok := os.LookupEnv("SECRET_KEY"); ok {
		cfg.SecretKey = secret
	}
	if lvl, ok := os.LookupEnv("LOG_LEVEL"); ok {
		cfg.LogLevel = lvl
	}
}
