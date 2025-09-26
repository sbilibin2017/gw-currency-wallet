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
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"github.com/sbilibin2017/gw-currency-wallet/internal/facades"
	"github.com/sbilibin2017/gw-currency-wallet/internal/handlers"

	"github.com/sbilibin2017/gw-currency-wallet/internal/jwt"
	"github.com/sbilibin2017/gw-currency-wallet/internal/logger"
	"github.com/sbilibin2017/gw-currency-wallet/internal/repositories"
	"github.com/sbilibin2017/gw-currency-wallet/internal/services"

	"github.com/sbilibin2017/gw-currency-wallet/internal/middlewares"

	_ "github.com/jackc/pgx/v5/stdlib"
	pb "github.com/sbilibin2017/proto-exchange/exchange"
	httpSwagger "github.com/swaggo/http-swagger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Build info variables
var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

// Package main gw-currency-wallet API.
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
		redisPoolSize, redisMinIdleConns, redisExp,
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
		redisPoolSize, redisMinIdleConns, redisExp,
		gwHost, gwPort,
		logLevel,
		jwtSecret, jwtExp,
	); err != nil {
		log.Fatalf("application stopped with error: %v", err)
	}
}

func printBuildInfo() {
	fmt.Printf("Version: %s\n", buildVersion)
	fmt.Printf("Commit: %s\n", buildCommit)
	fmt.Printf("Build: %s\n", buildDate)
}

func parseFlags() string {
	c := flag.String("c", "config.env", "Path to configuration file")
	flag.Parse()
	return *c
}

// parseConfig loads env and returns all configs including redisExp
func parseConfig(path string) (
	appHost, appPort string,
	pgHost string, pgPort int, pgUser, pgPassword, pgDB string,
	pgMaxOpenConns, pgMaxIdleConns int,
	redisHost string, redisPort, redisDB int, redisPassword string,
	redisPoolSize, redisMinIdleConns, redisExp int,
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

	// Application
	appHost = getEnv("APP_HOST", "localhost")
	appPort = getEnv("APP_PORT", "8080")
	logLevel = getEnv("APP_LOG_LEVEL", "info")

	// PostgreSQL
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

	// Redis
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
	if redisExp, err = strconv.Atoi(getEnv("REDIS_EXP_SECOND", "60")); err != nil {
		return
	}

	// gRPC
	gwHost = getEnv("GW_EXCHANGER_HOST", "localhost")
	gwPort = getEnv("GW_EXCHANGER_PORT", "50051")

	// JWT
	jwtSecretKey = getEnv("JWT_SECRET_KEY", "my_super_secret_key")
	if jwtExpSecond, err = strconv.Atoi(getEnv("JWT_EXP_SECOND", "60")); err != nil {
		return
	}

	return
}

// run initializes logger, DB, Redis, gRPC, JWT, services, handlers, router and handles shutdown
// run initializes logger, DB, Redis, gRPC, JWT, services, handlers, router, and handles graceful shutdown.
func run(ctx context.Context,
	appHost, appPort string,
	pgHost string, pgPort int, pgUser, pgPassword, pgDB string,
	pgMaxOpenConns, pgMaxIdleConns int,
	redisHost string, redisPort, redisDB int, redisPassword string,
	redisPoolSize, redisMinIdleConns, redisExp int,
	gwHost, gwPort, logLevel string,
	jwtSecretKey string, jwtExpSecond int,
) error {

	// Logger
	if err := logger.Initialize(logLevel); err != nil {
		fmt.Println("failed to initialize logger:", err)
		return err
	}
	defer logger.Log.Sync()
	logger.Log.Infof("Logger initialized with level %s", logLevel)

	// PostgreSQL
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		pgUser, pgPassword, pgHost, pgPort, pgDB)
	db, err := sqlx.ConnectContext(ctx, "pgx", dsn)
	if err != nil {
		logger.Log.Error("PostgreSQL connection error:", err)
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(pgMaxOpenConns)
	db.SetMaxIdleConns(pgMaxIdleConns)
	if err := db.PingContext(ctx); err != nil {
		logger.Log.Error("PostgreSQL ping failed:", err)
		return err
	}

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", redisHost, redisPort),
		Password:     redisPassword,
		DB:           redisDB,
		PoolSize:     redisPoolSize,
		MinIdleConns: redisMinIdleConns,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Log.Error("Redis connection error:", err)
		return err
	}
	defer rdb.Close()

	// gRPC client
	grpcAddr := fmt.Sprintf("%s:%s", gwHost, gwPort)
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Log.Error("Failed to connect to gRPC service at", grpcAddr, ":", err)
		return err
	}
	defer conn.Close()
	exchangeGRPCClient := pb.NewExchangeServiceClient(conn)

	// JWT
	jwtService := jwt.New(
		jwt.WithSecretKey(jwtSecretKey),
		jwt.WithExpiration(time.Duration(jwtExpSecond)*time.Second),
	)

	// Repositories
	userReadRepo := repositories.NewUserReadRepository(db)
	userWriteRepo := repositories.NewUserWriteRepository(db)
	walletReaderRepo := repositories.NewWalletReaderRepository(db)
	walletWriterRepo := repositories.NewWalletWriterRepository(db, nil)
	exchangeRateCacheRepo := repositories.NewExchangeRateCacheRepository(rdb, time.Duration(redisExp)*time.Second)
	exchangeGRPCFacade := facades.NewExchangeRatesGRPCFacade(exchangeGRPCClient)

	// Services
	authService := services.NewAuthService(userReadRepo, userWriteRepo, jwtService)
	walletService := services.NewWalletService(walletWriterRepo, walletReaderRepo, exchangeGRPCFacade, exchangeRateCacheRepo)

	// Handlers
	registerHandler := handlers.NewRegisterHandler(authService)
	loginHandler := handlers.NewLoginHandler(authService)
	balanceHandler := handlers.NewGetBalanceHandler(walletService, jwtService)
	depositHandler := handlers.NewDepositHandler(walletService, jwtService)
	withdrawHandler := handlers.NewWithdrawHandler(walletService, jwtService)
	getRatesHandler := handlers.NewGetExchangeRatesHandler(walletService, jwtService)
	exchangeHandler := handlers.NewExchangeHandler(jwtService, walletService)

	// Router
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middlewares.LoggingMiddleware)

	// Public routes
	r.Post("/register", registerHandler)
	r.Post("/login", loginHandler)

	// Authenticated routes
	authMiddleware := middlewares.AuthMiddleware(jwtService)
	txMiddleware := middlewares.TxMiddleware(db)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)

		r.Get("/balance", balanceHandler)
		r.With(txMiddleware).Post("/wallet/deposit", depositHandler)
		r.With(txMiddleware).Post("/wallet/withdraw", withdrawHandler)
		r.Get("/exchange/rates", getRatesHandler)
		r.With(txMiddleware).Post("/exchange", exchangeHandler)
	})

	// Swagger
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
		logger.Log.Infof("HTTP server listening on %s:%s", appHost, appPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("HTTP server failed: %w", err)
		}
	}()

	select {
	case <-ctxShutdown.Done():
		logger.Log.Info("Shutdown signal received, stopping HTTP server...")
	case serveErr := <-errChan:
		return serveErr
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Log.Errorw("HTTP server shutdown error", "error", err)
	}

	logger.Log.Info("HTTP server stopped gracefully")
	return nil
}
