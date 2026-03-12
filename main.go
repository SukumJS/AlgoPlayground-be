package main

import (
	"algoplayground/config"
	"algoplayground/routes"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {

	// load env
	config.LoadEnv()

	// production mode
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// init services
	config.InitFirebase()
	config.InitS3()

	// router
	r := gin.Default()

	// cors
	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{allowedOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Initialize routes
	routes.SetupRoutes(r)

	// Serve the inserter HTML tool
	r.Static("/inserter", "./inserter")

	// Start server on the port defined in .env or default to 8080
	port := os.Getenv("PORT")

	server := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// start server
	go func() {
		log.Println("Server running on port", port)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// graceful shutdown
	quit := make(chan os.Signal, 1)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
