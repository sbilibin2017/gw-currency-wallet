package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"github.com/sbilibin2017/gw-currency-wallet/internal/handlers"

	"github.com/sbilibin2017/gw-currency-wallet/internal/jwt"
	"github.com/sbilibin2017/gw-currency-wallet/internal/logger"
	"github.com/sbilibin2017/gw-currency-wallet/internal/repositories"
	"github.com/sbilibin2017/gw-currency-wallet/internal/services"

	"github.com/sbilibin2017/gw-currency-wallet/internal/middlewares"

	_ "github.com/jackc/pgx/v5/stdlib"
	// pb "github.com/sbilibin2017/proto-exchange/exchange"
	httpSwagger "github.com/swaggo/http-swagger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Build info variables, set via ldflags at build time.
var (
	buildVersion = "N/A" // Version of the service
	buildDate    = "N/A" // Build date
	buildCommit  = "N/A" // Git commit hash
)

// @title gw-currency-wallet API
// @version 1.0.0
// @description Microservice for managing user wallets and currency exchange
// @host localhost:8080
// @BasePath /api/v1
// @schemes http
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	printBuildInfo()
	configPath := parseFlags()

	appHost, appPort, pgHost, pgPort, pgUser, pgPassword, pgDB,
		pgMaxOpenConns, pgMaxIdleConns,
		redisHost, redisPort, redisDB, redisPassword,
		redisPoolSize, redisMinIdleConns,
		gwHost, gwPort, logLevel,
		jwtSecret, jwtExp,
		err := parseConfig(configPath)
	if err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	if err := run(context.Background(),
		appHost, appPort,
		pgHost, pgPort, pgUser, pgPassword, pgDB,
		pgMaxOpenConns, pgMaxIdleConns,
		redisHost, redisPort, redisDB, redisPassword,
		redisPoolSize, redisMinIdleConns,
		gwHost, gwPort,
		logLevel,
		jwtSecret, jwtExp,
	); err != nil {
		log.Fatalf("application stopped with error: %v", err)
	}
}

// printBuildInfo prints the build version, commit hash, and build date.
func printBuildInfo() {
	fmt.Printf("Starting service version %s, commit %s, build %s\n", buildVersion, buildCommit, buildDate)
}

// parseFlags parses command-line flags and returns the config file path.
func parseFlags() string {
	c := flag.String("c", "config.env", "Path to configuration file")
	flag.Parse()
	return *c
}

// parseConfig loads environment variables from a file and returns
// all application, database, Redis, gRPC, logging, and JWT configuration.
func parseConfig(path string) (
	appHost, appPort string,
	pgHost string, pgPort int, pgUser, pgPassword, pgDB string,
	pgMaxOpenConns, pgMaxIdleConns int,
	redisHost string, redisPort int, redisDB int, redisPassword string,
	redisPoolSize, redisMinIdleConns int,
	gwHost, gwPort, logLevel string,
	jwtSecretKey string, jwtExpSecond int,
	err error,
) {
	_ = godotenv.Load(path)

	getEnv := func(key, defaultValue string) string {
		if val, ok := os.LookupEnv(key); ok && val != "" {
			return val
		}
		return defaultValue
	}

	// Application config
	appHost = getEnv("APP_HOST", "localhost")
	appPort = getEnv("APP_PORT", "8080")
	logLevel = getEnv("APP_LOG_LEVEL", "info")

	// PostgreSQL config
	pgHost = getEnv("POSTGRES_HOST", "localhost")
	pgUser = getEnv("POSTGRES_USER", "user")
	pgPassword = getEnv("POSTGRES_PASSWORD", "password")
	pgDB = getEnv("POSTGRES_DB", "database")
	if pgPort, err = strconv.Atoi(getEnv("POSTGRES_PORT", "5432")); err != nil {
		return
	}
	if pgMaxOpenConns, err = strconv.Atoi(getEnv("POSTGRES_MAX_OPEN_CONNS", "16")); err != nil {
		return
	}
	if pgMaxIdleConns, err = strconv.Atoi(getEnv("POSTGRES_MAX_IDLE_CONNS", "8")); err != nil {
		return
	}

	// Redis config
	redisHost = getEnv("REDIS_HOST", "localhost")
	if redisPort, err = strconv.Atoi(getEnv("REDIS_PORT", "6379")); err != nil {
		return
	}
	if redisDB, err = strconv.Atoi(getEnv("REDIS_DB", "0")); err != nil {
		return
	}
	redisPassword = getEnv("REDIS_PASSWORD", "")
	if redisPoolSize, err = strconv.Atoi(getEnv("REDIS_POOL_SIZE", "10")); err != nil {
		return
	}
	if redisMinIdleConns, err = strconv.Atoi(getEnv("REDIS_MIN_IDLE_CONNS", "2")); err != nil {
		return
	}

	// gRPC config
	gwHost = getEnv("GW_EXCHANGER_HOST", "localhost")
	gwPort = getEnv("GW_EXCHANGER_PORT", "50051")

	// JWT config
	jwtSecretKey = getEnv("JWT_SECRET_KEY", "my_super_secret_key")
	if jwtExpSecond, err = strconv.Atoi(getEnv("JWT_EXP_SECOND", "60")); err != nil {
		return
	}

	return
}

// run initializes the logger, database, Redis, gRPC client, and HTTP server.
// It sets up routes, applies middleware, and handles graceful shutdown.
func run(ctx context.Context,
	appHost, appPort string,
	pgHost string, pgPort int, pgUser, pgPassword, pgDB string,
	pgMaxOpenConns, pgMaxIdleConns int,
	redisHost string, redisPort, redisDB int, redisPassword string,
	redisPoolSize, redisMinIdleConns int,
	gwHost, gwPort, logLevel string,
	jwtSecretKey string, jwtExpSecond int,
) error {
	// Initialize logger
	log, err := logger.New(logLevel)
	if err != nil {
		fmt.Println("failed to initialize logger:", err)
		return err
	}
	defer log.Sync()
	log.Infof("Logger initialized with level %s", logLevel)

	// Connect to PostgreSQL
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		pgUser, pgPassword, pgHost, pgPort, pgDB)
	log.Infof("Connecting to PostgreSQL: %s", dsn)

	db, err := sqlx.ConnectContext(ctx, "pgx", dsn)
	if err != nil {
		log.Fatal("PostgreSQL connection error:", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(pgMaxOpenConns)
	db.SetMaxIdleConns(pgMaxIdleConns)
	if err := db.PingContext(ctx); err != nil {
		log.Fatal("PostgreSQL ping failed:", err)
	}

	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", redisHost, redisPort),
		Password:     redisPassword,
		DB:           redisDB,
		PoolSize:     redisPoolSize,
		MinIdleConns: redisMinIdleConns,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal("Redis connection error:", err)
	}
	defer rdb.Close()

	// Connect to gRPC service
	grpcAddr := fmt.Sprintf("%s:%s", gwHost, gwPort)
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Failed to connect to gRPC service at", grpcAddr, ":", err)
	}
	defer conn.Close()
	// exchangeClient := pb.NewExchangeServiceClient(conn)

	// Initialize JWT service
	jwt := jwt.New(jwtSecretKey, time.Duration(jwtExpSecond)*time.Second)

	// Initialize repositories
	userReadRepo := repositories.NewUserReadRepository(db, log)
	userWriteRepo := repositories.NewUserWriteRepository(db, log)

	// Initialize services
	authService := services.NewAuthService(userReadRepo, userWriteRepo, jwt, log)

	// Initialize handlers
	registerHandler := handlers.NewRegisterHandler(authService, log)
	loginHandler := handlers.NewLoginHandler(authService, log)
	// balanceHandler := handlers.NewGetBalanceHandler()
	// depositHandler := handlers.NewDepositHandler()
	// withdrawHandler := handlers.NewWithdrawHandler()
	// getRatesHandler := handlers.NewGetRatesHandlerWithClient(exchangeClient)
	// exchangeHandler := handlers.NewExchangeHandlerWithClient(exchangeClient)

	// Setup router
	r := chi.NewRouter()
	r.Use(chimiddleware.Recoverer)
	r.Use(middlewares.LoggingMiddleware(log))

	// // Public routes
	r.Post("/register", registerHandler)
	r.Post("/login", loginHandler)

	// // Protected routes with JWT middleware
	// authMiddleware := middlewares.AuthMiddleware([]byte(jwtSecret))
	// r.Group(func(r chi.Router) {
	// 	r.Use(authMiddleware)
	// 	r.Get("/balance", balanceHandler)
	// 	r.Post("/wallet/deposit", depositHandler)
	// 	r.Post("/wallet/withdraw", withdrawHandler)
	// 	r.Get("/exchange/rates", getRatesHandler)
	// 	r.Post("/exchange", exchangeHandler)
	// })

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL(fmt.Sprintf("http://%s:%s/swagger/doc.json", appHost, appPort)),
	))

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", appHost, appPort),
		Handler: r,
	}

	// Graceful shutdown
	errChan := make(chan error, 1)
	ctxShutdown, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	go func() {
		log.Infof("HTTP server listening on %s:%s", appHost, appPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("HTTP server failed: %w", err)
		}
	}()

	select {
	case <-ctxShutdown.Done():
		log.Info("Shutdown signal received, stopping HTTP server...")
	case serveErr := <-errChan:
		return serveErr
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Errorw("HTTP server shutdown error", "error", err)
	}

	log.Info("HTTP server stopped gracefully")
	return nil
}
