package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"order-management/config"
	"order-management/controller"
	"order-management/database"
	"order-management/middleware"
	"order-management/repository"
	"order-management/service"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	var repo repository.OrderRepository
	var dbCloser func()
	if cfg.DBHost != "" {
		db, err := database.Connect(cfg)
		if err != nil {
			log.Fatalf("database connection failed: %v", err)
		}
		repo = repository.NewPostgresRepository(db)
		dbCloser = db.Close
	} else if cfg.Environment != "development" && cfg.Environment != "test" {
		log.Fatal("DB_HOST is required outside development/test mode")
	} else {
		log.Println("DB_HOST empty: starting with in-memory repository for local API exploration")
		repo = repository.NewMemoryRepository()
	}
	if dbCloser != nil {
		defer dbCloser()
	}

	orderService := service.NewOrderService(repo, service.NewHTTPProductClient(cfg.ProductServiceURL, cfg.ProductTimeout, cfg.ProductRetries), 0)
	orderController := controller.NewOrderController(orderService)
	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Recovery(), middleware.SecurityHeaders(), middleware.AccessLog(), middleware.Metrics())
	r.GET("/health/live", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "live"}) })
	r.GET("/health/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := repo.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "error": "database unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	r.GET("/metrics", middleware.PrometheusHandler())

	api := r.Group("/api/v1")
	if cfg.APIKey != "" {
		api.Use(middleware.APIKeyAuth(cfg.APIKey))
	}
	orderController.Register(api)
	controller.RegisterGraphQL(r, orderService)

	srv := &http.Server{Addr: ":" + cfg.AppPort, Handler: r, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("order API listening on :%s", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown error: %v", err)
	}
}
