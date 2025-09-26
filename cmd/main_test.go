package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	pb "github.com/sbilibin2017/proto-exchange/exchange"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/grpc"
)

// resetFlags resets the global flag.CommandLine to avoid "flag redefined" panic
func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

// resetEnv clears env vars used by parseConfig
func resetEnv() {
	os.Clearenv()
}

func TestParseFlags_Default(t *testing.T) {
	resetFlags()
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd"}
	configPath := parseFlags()
	expected := "config.env"

	if configPath != expected {
		t.Errorf("expected %s, got %s", expected, configPath)
	}
}

func TestParseFlags_Custom(t *testing.T) {
	resetFlags()
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd", "-c", "myconfig.env"}
	configPath := parseFlags()
	expected := "myconfig.env"

	if configPath != expected {
		t.Errorf("expected %s, got %s", expected, configPath)
	}
}

// ----------------- Tests for printBuildInfo -----------------

func TestPrintBuildInfo_Output(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set build info variables
	buildVersion = "v1.0.0"
	buildCommit = "abcd1234"
	buildDate = "2025-09-26"

	printBuildInfo()

	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()
	os.Stdout = oldStdout

	// Check if all expected strings are present
	if !contains(output, "Version: v1.0.0") ||
		!contains(output, "Commit: abcd1234") ||
		!contains(output, "Build: 2025-09-26") {
		t.Errorf("printBuildInfo output unexpected:\n%s", output)
	}
}

// Helper function to check substring
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
		gwHost, gwPort, logLevel,
		jwtSecret, jwtExp, err := parseConfig("nonexistent.env")

	if err != nil {
		t.Fatalf("parseConfig returned error: %v", err)
	}

	// Application
	if appHost != "localhost" || appPort != "8080" || logLevel != "info" {
		t.Errorf("unexpected app config: %v/%v/%v", appHost, appPort, logLevel)
	}

	// PostgreSQL
	if pgHost != "localhost" || pgPort != 5432 || pgUser != "user" || pgPassword != "password" || pgDB != "database" ||
		pgMaxOpenConns != 16 || pgMaxIdleConns != 8 {
		t.Errorf("unexpected postgres config")
	}

	// Redis
	if redisHost != "localhost" || redisPort != 6379 || redisDB != 0 || redisPassword != "" ||
		redisPoolSize != 10 || redisMinIdleConns != 2 || redisExp != 60 {
		t.Errorf("unexpected redis config")
	}

	// gRPC
	if gwHost != "localhost" || gwPort != "50051" {
		t.Errorf("unexpected grpc config")
	}

	// JWT
	if jwtSecret != "my_super_secret_key" || jwtExp != 60 {
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

	os.Setenv("JWT_SECRET_KEY", "supersecret")
	os.Setenv("JWT_EXP_SECOND", "300")

	appHost, appPort,
		pgHost, pgPort, pgUser, pgPassword, pgDB,
		pgMaxOpenConns, pgMaxIdleConns,
		redisHost, redisPort, redisDB, redisPassword,
		redisPoolSize, redisMinIdleConns, redisExp,
		gwHost, gwPort, logLevel,
		jwtSecret, jwtExp, err := parseConfig("nonexistent.env")

	if err != nil {
		t.Fatalf("parseConfig returned error: %v", err)
	}

	// Check all variables
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
	if jwtSecret != "supersecret" || jwtExp != 300 {
		t.Errorf("unexpected jwt config")
	}
}

// ------------------ Mock gRPC Server ------------------
type mockExchangeServer struct {
	pb.UnimplementedExchangeServiceServer
}

func (m *mockExchangeServer) GetExchangeRates(ctx context.Context, _ *pb.Empty) (*pb.ExchangeRatesResponse, error) {
	return &pb.ExchangeRatesResponse{
		Rates: map[string]float32{
			"USD": 1.0,
			"EUR": 0.9,
			"JPY": 110.0,
		},
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
	return &pb.ExchangeRateResponse{
		FromCurrency: req.FromCurrency,
		ToCurrency:   req.ToCurrency,
		Rate:         rate,
	}, nil
}

// Start mock gRPC server and return host:port and stop function
func startMockGRPCServer() (addr string, stop func(), err error) {
	lis, err := net.Listen("tcp", "127.0.0.1:0") // OS assigns a free port
	if err != nil {
		return "", nil, err
	}
	s := grpc.NewServer()
	pb.RegisterExchangeServiceServer(s, &mockExchangeServer{})
	go s.Serve(lis)

	stop = func() {
		s.Stop()
		lis.Close()
	}
	return lis.Addr().String(), stop, nil
}

// ------------------ Full integration test ------------------
func TestRun_Success(t *testing.T) {
	ctx := context.Background()

	// ------------------ Postgres container ------------------
	pgReq := testcontainers.ContainerRequest{
		Image:        "postgres:15",
		Env:          map[string]string{"POSTGRES_PASSWORD": "password", "POSTGRES_DB": "testdb", "POSTGRES_USER": "user"},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp"),
	}
	pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: pgReq, Started: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pgContainer.Terminate(ctx)

	pgHost, _ := pgContainer.Host(ctx)
	pgPort, _ := pgContainer.MappedPort(ctx, "5432")

	// ------------------ Redis container ------------------
	redisReq := testcontainers.ContainerRequest{
		Image:        "redis:7",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp"),
	}
	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: redisReq, Started: true})
	if err != nil {
		t.Fatal(err)
	}
	defer redisContainer.Terminate(ctx)

	redisHost, _ := redisContainer.Host(ctx)
	redisPort, _ := redisContainer.MappedPort(ctx, "6379")

	// ------------------ Mock gRPC server ------------------
	grpcAddr, stopGRPC, err := startMockGRPCServer()
	if err != nil {
		t.Fatal(err)
	}
	defer stopGRPC()

	// Split host and port
	var grpcHost, grpcPort string
	fmt.Sscanf(grpcAddr, "%s:%s", &grpcHost, &grpcPort)

	// ------------------ Run ------------------
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- run(testCtx,
			"127.0.0.1", "8086", // appHost, appPort
			pgHost, pgPort.Int(), "user", "password", "testdb",
			5, 2, // Postgres max connections
			redisHost, redisPort.Int(), 0, "", 10, 2, 60, // Redis
			grpcHost, grpcPort, // gRPC
			"debug",          // logLevel
			"testsecret", 60, // JWT
		)
	}()

	select {
	case <-time.After(11 * time.Second):
		t.Fatal("test timed out")
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected run to succeed, got error: %v", err)
		}
		t.Log("run completed successfully")
	}
}
