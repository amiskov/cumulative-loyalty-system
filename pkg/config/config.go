package config

import (
	"flag"
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	RunAddress             string
	DatabaseURI            string
	AccrualSystemAddress   string        // full address, like `http//localhost:8888`
	AccrualPollingLimit    int           // max attempts to get order info from accrual system
	AccrualPollingInterval time.Duration // pause between attempts to get an order info from accrual
	AccrualRequestTimeout  time.Duration
	LogLevel               string
	SecretKey              string
}

func Parse() *Config {
	cfg := Config{
		// Defaults
		RunAddress:             "localhost:8080",
		AccrualSystemAddress:   "http://localhost:8888",
		AccrualPollingLimit:    100,
		AccrualPollingInterval: 1 * time.Second,
		AccrualRequestTimeout:  3 * time.Second,
		SecretKey:              "secret",
		LogLevel:               "debug",
	}
	cfg.updateFromFlags()
	cfg.updateFromEnv()
	return &cfg
}

func (cfg *Config) updateFromFlags() {
	flagRunAddress := flag.String("a", cfg.RunAddress, "Server address.")
	flagDatabaseURI := flag.String("d", cfg.DatabaseURI, "Postgres DSN.")
	flagAccrualAddress := flag.String("r", cfg.AccrualSystemAddress, "Accrual System address.")
	flagAccrualPollingInterval := flag.Duration("i", cfg.AccrualPollingInterval,
		"Pause between attempts to get order accrual.")
	flagAccrualPollingLimit := flag.Int("l", cfg.AccrualPollingLimit,
		"Max attempts to get order accrual from the accrual system.")
	flagAccrualRequestTimeout := flag.Duration("t", cfg.AccrualPollingInterval,
		"Pause between attempts to get order accrual.")

	flag.Parse()

	cfg.RunAddress = *flagRunAddress
	cfg.DatabaseURI = *flagDatabaseURI
	cfg.AccrualSystemAddress = *flagAccrualAddress
	cfg.AccrualPollingLimit = *flagAccrualPollingLimit
	cfg.AccrualPollingInterval = *flagAccrualPollingInterval
	cfg.AccrualRequestTimeout = *flagAccrualRequestTimeout
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
	if interval, ok := os.LookupEnv("ACCRUAL_POLLING_INTERVAL"); ok {
		i, err := strconv.Atoi(interval)
		if err != nil {
			log.Fatal("bad accrual polling interval value, must be int (seconds)")
		}
		cfg.AccrualPollingInterval = time.Duration(i) * time.Second
	}
	if timeout, ok := os.LookupEnv("ACCRUAL_REQUEST_TIMEOUT"); ok {
		t, err := strconv.Atoi(timeout)
		if err != nil {
			log.Fatal("bad accrual request timeout value, must be int (seconds)")
		}
		cfg.AccrualRequestTimeout = time.Duration(t) * time.Second
	}
	if secret, ok := os.LookupEnv("SECRET_KEY"); ok {
		cfg.SecretKey = secret
	}
	if lvl, ok := os.LookupEnv("LOG_LEVEL"); ok {
		cfg.LogLevel = lvl
	}
}
