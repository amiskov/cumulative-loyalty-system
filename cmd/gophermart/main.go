package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/joho/godotenv"

	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
	"github.com/amiskov/cumulative-loyalty-system/pkg/middleware"
	"github.com/amiskov/cumulative-loyalty-system/pkg/sessions"
	"github.com/amiskov/cumulative-loyalty-system/pkg/user"
	"github.com/amiskov/cumulative-loyalty-system/pkg/user/api"
)

type EnvConfig map[string]string

func main() {
	var cfg EnvConfig = readDotenv()
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db, err := sql.Open("pgx", "postgresql://localhost/"+cfg["POSTGRES_DB"]+"?sslmode=disable")
	if err != nil {
		log.Printf("Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("unable to reach PostgreSQL: %v", err)
	}

	usersRepo := user.NewUserRepo(db)
	sessionRepo := sessions.NewSessionRepo(db)
	sessionManager := sessions.NewSessionManager(cfg["SECRET_KEY"], sessionRepo)
	userHandler := api.NewUserHandler(usersRepo, sessionManager)
	r := mux.NewRouter()

	api := r.PathPrefix("/api").Subrouter()

	// User
	api.HandleFunc("/user/register", userHandler.Register).Methods("POST")
	api.HandleFunc("/user/login", userHandler.LogIn).Methods("POST")

	// auth := middleware.NewAuthMiddleware(sessionManager, usersRepo)
	// r.Use(auth.Middleware)

	logMiddleware := middleware.NewLoggingMiddleware(logger.Run(cfg["LOG_LEVEL"]))
	r.Use(logMiddleware.SetupTracing)
	r.Use(logMiddleware.SetupLogging)
	r.Use(logMiddleware.AccessLog)

	log.Println("Serving at http://localhost:8080/")
	log.Fatalln(http.ListenAndServe(":8080", r))

}

func readDotenv() EnvConfig {
	env, err := godotenv.Read()
	if err != nil {
		log.Fatal("failed reading .env file:", err)
	}
	return env
}
