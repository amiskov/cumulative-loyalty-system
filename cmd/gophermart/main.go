package main

import (
	"database/sql"
	"log"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/gorilla/mux"
	_ "github.com/jackc/pgx/v4/stdlib"

	"github.com/amiskov/cumulative-loyalty-system/pkg/balance"
	balanceApi "github.com/amiskov/cumulative-loyalty-system/pkg/balance/api"
	"github.com/amiskov/cumulative-loyalty-system/pkg/config"
	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
	"github.com/amiskov/cumulative-loyalty-system/pkg/middleware"
	"github.com/amiskov/cumulative-loyalty-system/pkg/order"
	orderApi "github.com/amiskov/cumulative-loyalty-system/pkg/order/api"
	"github.com/amiskov/cumulative-loyalty-system/pkg/sessions"
	"github.com/amiskov/cumulative-loyalty-system/pkg/user"
	userApi "github.com/amiskov/cumulative-loyalty-system/pkg/user/api"
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

	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatalln("can't create cookie jar")
	}

	httpClient := resty.New().
		SetHostURL(cfg.AccrualSystemAddress).
		SetCookieJar(jar)

	sessionRepo := sessions.NewSessionRepo(db)
	sessionManager := sessions.NewSessionManager(cfg.SecretKey, sessionRepo)

	usersRepo := user.NewUserRepo(db)
	userHandler := userApi.NewUserHandler(usersRepo, sessionManager)

	ordersRepo := order.NewOrderRepo(db)
	orderHandler := orderApi.NewOrderHandler(ordersRepo, httpClient)

	balanceRepo := balance.NewBalanceRepo(db)
	balanceHandler := balanceApi.NewBalanceHandler(balanceRepo, httpClient)

	r := mux.NewRouter()

	api := r.PathPrefix("/api").Subrouter()

	// User
	api.HandleFunc("/user/register", userHandler.Register).Methods("POST")
	api.HandleFunc("/user/login", userHandler.LogIn).Methods("POST")

	// Order
	api.HandleFunc("/user/orders", orderHandler.AddOrder).Methods("POST")
	api.HandleFunc("/user/orders", orderHandler.GetOrdersList).Methods("GET")

	// Balance
	api.HandleFunc("/user/balance", balanceHandler.GetUserBalance).Methods("GET")
	api.HandleFunc("/user/balance/withdraw", balanceHandler.Withdraw).Methods("POST")
	api.HandleFunc("/user/withdrawals", balanceHandler.Withdrawalls).Methods("GET")

	auth := middleware.NewAuthMiddleware(sessionManager, usersRepo)
	r.Use(auth.Middleware)

	logMiddleware := middleware.NewLoggingMiddleware(logger.Run(cfg.LogLevel))
	r.Use(logMiddleware.SetupTracing)
	r.Use(logMiddleware.SetupLogging)
	r.Use(logMiddleware.AccessLog)

	log.Println("Serving at http://localhost:8080/")
	log.Fatalln(http.ListenAndServe(":8080", r))
}
