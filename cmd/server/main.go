package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"genai-processing/internal/processor"
)

func main() {
	// Initialize logger
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting GenAI Audit Query Processor Server")

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize GenAI processor
	log.Println("Initializing GenAI processor...")
	genaiProcessor := processor.NewGenAIProcessor()
	if genaiProcessor == nil {
		log.Fatal("Failed to initialize GenAI processor")
	}
	log.Println("✓ GenAI processor initialized successfully")

	// Setup routes
	log.Println("Setting up HTTP routes...")
	mux := setupRoutes(genaiProcessor)
	log.Println("✓ HTTP routes configured")

	// Add middleware
	log.Println("Configuring middleware...")
	handler := corsMiddleware(loggingMiddleware(mux))
	log.Println("✓ Middleware configured")

	// Configure server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Log startup completion
	log.Printf("✓ Server started successfully on port %s", port)
	log.Println("✓ POST /query - Process natural language audit queries")
	log.Println("✓ GET  /health - Health check endpoint")
	log.Println("Press Ctrl+C to shutdown gracefully")

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create a deadline for server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}
