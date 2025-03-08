package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashperp/hashperp"
	"github.com/hashperp/hashperp/api"
	"github.com/hashperp/hashperp/bitcoin"
	"github.com/hashperp/hashperp/storage"
	
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or could not be loaded: %v", err)
	}
	
	// Initialize logger
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	
	log.Println("Starting HashPerp service...")
	
	// Connect to database
	db, err := connectToDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	
	// Run database migrations
	if err := storage.MigrateDB(db); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}
	
	// Initialize Bitcoin client
	btcClient, err := initializeBitcoinClient()
	if err != nil {
		log.Fatalf("Failed to initialize Bitcoin client: %v", err)
	}
	
	// Initialize repositories
	contractRepo := storage.NewPostgresContractRepository(db)
	vtxoRepo := storage.NewPostgresVTXORepository(db)
	orderRepo := storage.NewPostgresOrderRepository(db)
	swapOfferRepo := storage.NewPostgresSwapOfferRepository(db)
	transactionRepo := storage.NewPostgresTransactionRepository(db)
	hashRateRepo := storage.NewPostgresHashRateRepository(db)
	userRepo := storage.NewPostgresUserRepository(db)
	
	// Initialize script generator
	scriptGen := hashperp.NewScriptGeneratorService(btcClient, 144) // 144 blocks timeout (approx. 1 day)
	
	// Initialize managers/services
	// Note the cyclic dependency between services, we need to create them first then set dependencies
	marketDataMgr := hashperp.NewMarketDataManager(hashRateRepo, btcClient)
	transactionMgr := hashperp.NewTransactionManager(transactionRepo)
	
	// Create VTXO manager and swap offer manager with nil dependencies for now
	vtxoMgr := hashperp.NewVTXOService(vtxoRepo, contractRepo, transactionRepo, scriptGen, btcClient)
	swapOfferMgr := hashperp.NewSwapOfferService(swapOfferRepo, vtxoRepo, contractRepo, transactionRepo, nil)
	
	// Now set the VTXOManager in the SwapOfferManager
	if swapOfferSetter, ok := swapOfferMgr.(interface{ SetVTXOManager(hashperp.VTXOManager) }); ok {
		swapOfferSetter.SetVTXOManager(vtxoMgr)
	}
	
	// Create contract manager
	contractMgr := hashperp.NewContractService(contractRepo, vtxoRepo, transactionRepo, scriptGen, btcClient, swapOfferMgr)
	
	// Create order book manager
	orderBookMgr := hashperp.NewOrderBookService(orderRepo, contractRepo, contractMgr, transactionRepo, btcClient)
	
	// Create the main service
	service := hashperp.NewHashPerpService(
		contractMgr,
		vtxoMgr,
		orderBookMgr,
		swapOfferMgr,
		marketDataMgr,
		transactionMgr,
		scriptGen,
		btcClient,
	)
	
	// Initialize API server
	apiServer := api.NewServer(service)
	
	// Start API server
	go func() {
		addr := getEnv("API_ADDR", ":8080")
		log.Printf("Starting API server on %s", addr)
		if err := apiServer.Start(addr); err != nil {
			log.Fatalf("API server error: %v", err)
		}
	}()
	
	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	
	<-quit
	log.Println("Shutting down HashPerp service...")
	
	// Create a context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Shutdown API server
	if err := apiServer.Shutdown(ctx); err != nil {
		log.Fatalf("API server shutdown error: %v", err)
	}
	
	log.Println("HashPerp service shutdown complete")
}

// connectToDatabase establishes a connection to the PostgreSQL database
func connectToDatabase() (*gorm.DB, error) {
	dsn := getEnv("DATABASE_URL", "host=localhost user=postgres password=postgres dbname=hashperp port=5432 sslmode=disable")
	
	// Configure GORM logger
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: 200 * time.Millisecond,
			LogLevel:      logger.Warn,
			Colorful:      true,
		},
	)
	
	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}
	
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	
	return db, nil
}

// initializeBitcoinClient creates and configures the Bitcoin client
func initializeBitcoinClient() (hashperp.BitcoinClient, error) {
	rpcURL := getEnv("BITCOIN_RPC_URL", "http://localhost:8332")
	rpcUser := getEnv("BITCOIN_RPC_USER", "bitcoinrpc")
	rpcPassword := getEnv("BITCOIN_RPC_PASSWORD", "")
	
	// Create Bitcoin client
	return bitcoin.NewBitcoinClient(rpcURL, rpcUser, rpcPassword)
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}