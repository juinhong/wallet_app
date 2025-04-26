package main

import (
	"database/sql"
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	goredis "github.com/redis/go-redis/v9"

	"Crypto.com/internal/config"
	"Crypto.com/internal/handlers"
	"Crypto.com/internal/repositories/postgres"
	"Crypto.com/internal/repositories/redis"
	"Crypto.com/internal/services"
	"Crypto.com/pkg/utils"
)

func main() {
	cfg := config.LoadConfig()
	utils.Init(cfg.Environment == "production", cfg.LogPath)

	// Initialize PostgreSQL
	connStr := "postgres://" + cfg.DBUser + ":" + cfg.DBPassword + "@" + cfg.DBHost + ":" + cfg.DBPort + "/" + cfg.DBName
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatal("Error connecting to PostgreSQL:", err)
	}
	defer db.Close()

	// Initialize Redis
	redisClient := goredis.NewClient(&goredis.Options{
		Addr:     cfg.RedisHost + ":" + strconv.Itoa(cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// Initialize services
	walletRepo := postgres.NewWalletRepository(db, utils.Log)
	cacheRepo := redis.NewCacheRepository(redisClient, time.Hour, utils.Log)
	walletService := services.NewWalletService(walletRepo, cacheRepo, utils.Log)
	walletHandler := handlers.NewWalletHandler(walletService)

	// Create router
	router := gin.Default()
	router.Use(gin.Recovery())
	router.Use(handlers.LoggingHandler(utils.Log))

	// Wallet routes
	v1 := router.Group("/api/v1")
	{
		wallets := v1.Group("/wallets")
		wallets.POST("/:userID/deposit", walletHandler.Deposit)
		wallets.POST("/:userID/withdraw", walletHandler.Withdraw)
		wallets.POST("/:userID/transfer", walletHandler.Transfer)
		wallets.GET("/:userID/balance", walletHandler.GetBalance)
		wallets.GET("/:userID/transactions", walletHandler.TransactionHistory)
	}

	// Start server
	port := ":" + cfg.ServerPort
	log.Printf("Server starting on port %s", port)
	log.Fatal(router.Run(port))
}
