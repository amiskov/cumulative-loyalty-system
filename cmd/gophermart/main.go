package main

import (
	"database/sql"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/gorilla/mux"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/amiskov/cumulative-loyalty-system/pkg/accrual"
	"github.com/amiskov/cumulative-loyalty-system/pkg/balance"
	"github.com/amiskov/cumulative-loyalty-system/pkg/config"
	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
	"github.com/amiskov/cumulative-loyalty-system/pkg/middleware"
	"github.com/amiskov/cumulative-loyalty-system/pkg/order"
	"github.com/amiskov/cumulative-loyalty-system/pkg/session"
	"github.com/amiskov/cumulative-loyalty-system/pkg/user"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	cfg := config.Parse()

	db, openDBErr := sql.Open("pgx", cfg.DatabaseURI)
	if openDBErr != nil {
		log.Printf("Unable to connect to database: %v\n", openDBErr)
		os.Exit(1)
	}
	if pingErr := db.Ping(); pingErr != nil {
		log.Fatalf("unable to reach PostgreSQL: %v", pingErr)
	}
	defer db.Close()

	if err := migrateDB(db); err != nil {
		log.Fatal("can't migrate db", err)
	}

	userRepo := user.NewRepo(db)
	orderRepo := order.NewRepo(db)
	balanceRepo := balance.NewRepo(db)
	sessionRepo := session.NewSessionRepo(db)

	accrualClient := accrual.NewHTTPClient(cfg.AccrualSystemAddress,
		cfg.AccrualPollingLimit, cfg.AccrualRequestTimeout, cfg.AccrualPollingInterval)

	sessionService := session.NewSessionService(cfg.SecretKey, sessionRepo)
	orderService := order.NewService(orderRepo, accrualClient)
	userService := user.NewService(userRepo, sessionService)
	balanceService := balance.NewService(balanceRepo)

	userHandler := user.NewHandler(userService)
	orderHandler := order.NewOrderHandler(orderService)
	balanceHandler := balance.NewBalanceHandler(balanceService)

	r := mux.NewRouter()
	api := r.PathPrefix("/api").Subrouter()

	// User
	api.HandleFunc("/user/register", userHandler.Register).Methods("POST")
	api.HandleFunc("/user/login", userHandler.LogIn).Methods("POST")
	api.HandleFunc("/user/logout", userHandler.LogOut).Methods("POST")

	// Order
	api.HandleFunc("/user/orders", orderHandler.AddOrder).Methods("POST")
	api.HandleFunc("/user/orders", orderHandler.GetOrdersList).Methods("GET")

	// Balance
	api.HandleFunc("/user/balance", balanceHandler.GetUserBalance).Methods("GET")
	api.HandleFunc("/user/balance/withdraw", balanceHandler.Withdraw).Methods("POST")
	api.HandleFunc("/user/withdrawals", balanceHandler.Withdrawals).Methods("GET")

	noAuthUrls := map[string]struct{}{
		"/api/user/login":    {},
		"/api/user/register": {},
	}
	auth := middleware.NewAuthMiddleware(sessionService, userRepo, noAuthUrls)
	r.Use(auth.Middleware)

	logMiddleware := middleware.NewLoggingMiddleware(logger.Run(cfg.LogLevel))
	r.Use(logMiddleware.SetupTracing)
	r.Use(logMiddleware.SetupLogging)
	r.Use(logMiddleware.AccessLog)

	server := &http.Server{
		Addr:              cfg.RunAddress,
		Handler:           r,
		ReadHeaderTimeout: 2 * time.Second,
	}
	log.Println("Serving at http://" + cfg.RunAddress + "/")
	log.Fatalln(server.ListenAndServe())
}

func migrateDB(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}
	migrationsPath, err := filepath.Abs("migrations")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithDatabaseInstance("file:///"+migrationsPath, "postgres", driver)
	if err != nil {
		return err
	}
	m.Up()
	return nil
}
