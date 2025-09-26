package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/sbilibin2017/gw-currency-wallet/internal/logger"
	pb "github.com/sbilibin2017/proto-exchange/exchange"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/grpc"
)

// ------------------ Helper functions ------------------

func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

func resetEnv() {
	os.Clearenv()
}

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}

func TestParseConfig_Defaults(t *testing.T) {
	resetEnv()

	appHost, appPort,
		pgHost, pgPort, pgUser, pgPassword, pgDB,
		pgMaxOpenConns, pgMaxIdleConns,
		redisHost, redisPort, redisDB, redisPassword,
		redisPoolSize, redisMinIdleConns, redisExp,
		gwHost, gwPort,
		kafkaBrokers, kafkaTopic,
		logLevel,
		jwtSecretKey, jwtExpSecond, err := parseConfig("nonexistent.env")

	if err != nil {
		t.Fatalf("parseConfig returned error: %v", err)
	}

	// Application defaults
	if appHost != "localhost" || appPort != "8080" || logLevel != "info" {
		t.Errorf("unexpected app config: %v/%v/%v", appHost, appPort, logLevel)
	}

	// PostgreSQL defaults
	if pgHost != "localhost" || pgPort != 5432 || pgUser != "user" || pgPassword != "password" || pgDB != "database" ||
		pgMaxOpenConns != 16 || pgMaxIdleConns != 8 {
		t.Errorf("unexpected postgres config")
	}

	// Redis defaults
	if redisHost != "localhost" || redisPort != 6379 || redisDB != 0 || redisPassword != "" ||
		redisPoolSize != 10 || redisMinIdleConns != 2 || redisExp != 60 {
		t.Errorf("unexpected redis config")
	}

	// gRPC defaults
	if gwHost != "localhost" || gwPort != "50051" {
		t.Errorf("unexpected grpc config")
	}

	// Kafka defaults
	if !reflect.DeepEqual(kafkaBrokers, []string{"localhost:9092"}) || kafkaTopic != "large-transactions" {
		t.Errorf("unexpected kafka config: %v/%v", kafkaBrokers, kafkaTopic)
	}

	// JWT defaults
	if jwtSecretKey != "my_super_secret_key" || jwtExpSecond != 60 {
		t.Errorf("unexpected jwt config")
	}
}

func TestParseConfig_CustomEnv(t *testing.T) {
	resetEnv()

	os.Setenv("APP_HOST", "127.0.0.1")
	os.Setenv("APP_PORT", "9090")
	os.Setenv("APP_LOG_LEVEL", "debug")

	os.Setenv("POSTGRES_HOST", "pg.example.com")
	os.Setenv("POSTGRES_PORT", "5433")
	os.Setenv("POSTGRES_USER", "admin")
	os.Setenv("POSTGRES_PASSWORD", "secret")
	os.Setenv("POSTGRES_DB", "mydb")
	os.Setenv("POSTGRES_MAX_OPEN_CONNS", "20")
	os.Setenv("POSTGRES_MAX_IDLE_CONNS", "10")

	os.Setenv("REDIS_HOST", "redis.example.com")
	os.Setenv("REDIS_PORT", "6380")
	os.Setenv("REDIS_DB", "2")
	os.Setenv("REDIS_PASSWORD", "redispass")
	os.Setenv("REDIS_POOL_SIZE", "15")
	os.Setenv("REDIS_MIN_IDLE_CONNS", "5")
	os.Setenv("REDIS_EXP_SECOND", "120")

	os.Setenv("GW_EXCHANGER_HOST", "grpc.example.com")
	os.Setenv("GW_EXCHANGER_PORT", "50052")

	os.Setenv("KAFKA_BROKERS", "broker1:9092,broker2:9093")
	os.Setenv("KAFKA_TOPIC", "custom-topic")

	os.Setenv("JWT_SECRET_KEY", "supersecret")
	os.Setenv("JWT_EXP_SECOND", "300")

	appHost, appPort,
		pgHost, pgPort, pgUser, pgPassword, pgDB,
		pgMaxOpenConns, pgMaxIdleConns,
		redisHost, redisPort, redisDB, redisPassword,
		redisPoolSize, redisMinIdleConns, redisExp,
		gwHost, gwPort,
		kafkaBrokers, kafkaTopic,
		logLevel,
		jwtSecretKey, jwtExpSecond, err := parseConfig("nonexistent.env")

	if err != nil {
		t.Fatalf("parseConfig returned error: %v", err)
	}

	// Assertions
	if appHost != "127.0.0.1" || appPort != "9090" || logLevel != "debug" {
		t.Errorf("unexpected app config")
	}

	if pgHost != "pg.example.com" || pgPort != 5433 || pgUser != "admin" || pgPassword != "secret" || pgDB != "mydb" ||
		pgMaxOpenConns != 20 || pgMaxIdleConns != 10 {
		t.Errorf("unexpected postgres config")
	}

	if redisHost != "redis.example.com" || redisPort != 6380 || redisDB != 2 || redisPassword != "redispass" ||
		redisPoolSize != 15 || redisMinIdleConns != 5 || redisExp != 120 {
		t.Errorf("unexpected redis config")
	}

	if gwHost != "grpc.example.com" || gwPort != "50052" {
		t.Errorf("unexpected grpc config")
	}

	expectedBrokers := []string{"broker1:9092", "broker2:9093"}
	if !reflect.DeepEqual(kafkaBrokers, expectedBrokers) || kafkaTopic != "custom-topic" {
		t.Errorf("unexpected kafka config: %v/%v", kafkaBrokers, kafkaTopic)
	}

	if jwtSecretKey != "supersecret" || jwtExpSecond != 300 {
		t.Errorf("unexpected jwt config")
	}
}

// ------------------ Mock gRPC Server ------------------

type mockExchangeServer struct {
	pb.UnimplementedExchangeServiceServer
}

func (m *mockExchangeServer) GetExchangeRates(ctx context.Context, _ *pb.Empty) (*pb.ExchangeRatesResponse, error) {
	return &pb.ExchangeRatesResponse{
		Rates: map[string]float32{"USD": 1.0, "EUR": 0.9, "JPY": 110.0},
	}, nil
}

func (m *mockExchangeServer) GetExchangeRateForCurrency(ctx context.Context, req *pb.CurrencyRequest) (*pb.ExchangeRateResponse, error) {
	rate := float32(1.0)
	switch req.ToCurrency {
	case "EUR":
		rate = 0.9
	case "JPY":
		rate = 110.0
	}
	return &pb.ExchangeRateResponse{FromCurrency: req.FromCurrency, ToCurrency: req.ToCurrency, Rate: rate}, nil
}

func startMockGRPCServer(t *testing.T) (addr string, stop func()) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := grpc.NewServer()
	pb.RegisterExchangeServiceServer(s, &mockExchangeServer{})
	go s.Serve(lis)

	return lis.Addr().String(), func() {
		s.Stop()
		lis.Close()
	}
}

// ------------------ Unit tests ------------------

func TestParseFlags_Default(t *testing.T) {
	resetFlags()
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd"}
	configPath := parseFlags()
	if configPath != "config.env" {
		t.Errorf("expected config.env, got %s", configPath)
	}
}

func TestParseFlags_Custom(t *testing.T) {
	resetFlags()
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd", "-c", "myconfig.env"}
	configPath := parseFlags()
	if configPath != "myconfig.env" {
		t.Errorf("expected myconfig.env, got %s", configPath)
	}
}

func TestPrintBuildInfo_Output(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	buildVersion = "v1.0.0"
	buildCommit = "abcd1234"
	buildDate = "2025-09-26"

	printBuildInfo()
	w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = oldStdout

	output := buf.String()
	if !contains(output, "Version: v1.0.0") ||
		!contains(output, "Commit: abcd1234") ||
		!contains(output, "Build: 2025-09-26") {
		t.Errorf("printBuildInfo output unexpected:\n%s", output)
	}
}

// ------------------ Full Integration Test ------------------

func TestRun_FullIntegration(t *testing.T) {
	ctx := context.Background()
	logger.Initialize("debug")

	// ------------------ PostgreSQL ------------------
	pgReq := testcontainers.ContainerRequest{
		Image:        "postgres:15",
		Env:          map[string]string{"POSTGRES_PASSWORD": "password", "POSTGRES_DB": "testdb", "POSTGRES_USER": "user"},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}
	pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: pgReq, Started: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pgContainer.Terminate(ctx)

	pgHost, _ := pgContainer.Host(ctx)
	pgPortNat, _ := pgContainer.MappedPort(ctx, "5432")
	pgPort := pgPortNat.Int()

	// ------------------ Redis ------------------
	redisReq := testcontainers.ContainerRequest{
		Image:        "redis:7",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(30 * time.Second),
	}
	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: redisReq, Started: true})
	if err != nil {
		t.Fatal(err)
	}
	defer redisContainer.Terminate(ctx)

	redisHost, _ := redisContainer.Host(ctx)
	redisPortNat, _ := redisContainer.MappedPort(ctx, "6379")
	redisPort := redisPortNat.Int()

	// ------------------ Mock gRPC ------------------
	grpcAddr, stopGRPC := startMockGRPCServer(t)
	defer stopGRPC()
	var grpcHost, grpcPort string
	fmt.Sscanf(grpcAddr, "%s:%s", &grpcHost, &grpcPort)

	// ------------------ Run Application ------------------
	runCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- run(runCtx,
			"127.0.0.1", "8086", // HTTP
			pgHost, pgPort, "user", "password", "testdb",
			5, 2, // Postgres max connections
			redisHost, redisPort, 0, "", 10, 2, 60, // Redis
			grpcHost, grpcPort, // gRPC
			[]string{"localhost:9092"}, "large-transactions", // Kafka (not tested)
			"debug",
			"testsecret", 60,
		)
	}()

	// Wait a few seconds to let the server start
	time.Sleep(5 * time.Second)

	// ------------------ Shutdown ------------------
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run() returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for app shutdown")
	}
}
