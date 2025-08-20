package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"genai-processing/internal/config"
	"genai-processing/internal/processor"
)

func main() {
	// Initialize logger
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting GenAI Audit Query Processor Server")

	// Load configuration at startup
	log.Println("Loading configuration...")
	appConfig, err := loadConfiguration()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	log.Println("✓ Configuration loaded successfully")

	// Log configuration status during startup
	logConfigurationStatus(appConfig)

	// Initialize GenAI processor with configuration
	log.Println("Initializing GenAI processor with configuration...")
	genaiProcessor, err := initializeProcessor(appConfig)
	if err != nil {
		log.Fatalf("Failed to initialize GenAI processor: %v", err)
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

	// Configure server using loaded configuration
	server := &http.Server{
		Addr:         appConfig.Server.Host + ":" + appConfig.Server.Port,
		Handler:      handler,
		ReadTimeout:  appConfig.Server.ReadTimeout,
		WriteTimeout: appConfig.Server.WriteTimeout,
		IdleTimeout:  appConfig.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on %s:%s", appConfig.Server.Host, appConfig.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Log startup completion
	log.Printf("✓ Server started successfully on %s:%s", appConfig.Server.Host, appConfig.Server.Port)
	log.Printf("✓ Server timeouts - Read: %v, Write: %v, Idle: %v",
		appConfig.Server.ReadTimeout, appConfig.Server.WriteTimeout, appConfig.Server.IdleTimeout)
	log.Printf("✓ Default LLM provider: %s", appConfig.Models.DefaultProvider)
	log.Println("✓ POST /query - Process natural language audit queries")
	log.Println("✓ GET  /health - Health check endpoint")
	log.Println("✓ GET  /openapi.json - OpenAPI specification")
	log.Println("✓ GET  /docs - Swagger UI documentation")
	log.Println("✓ GET  /redoc - ReDoc documentation")
	log.Println("Press Ctrl+C to shutdown gracefully")

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create a deadline for server shutdown using configuration
	ctx, cancel := context.WithTimeout(context.Background(), appConfig.Server.ShutdownTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}

// loadConfiguration loads the application configuration using config.LoadConfig()
func loadConfiguration() (*config.AppConfig, error) {
	// Determine config directory
	configDir := os.Getenv("CONFIG_DIR")
	if configDir == "" {
		// Default to configs directory relative to executable
		execPath, err := os.Executable()
		if err != nil {
			// Fallback to current directory
			configDir = "configs"
		} else {
			configDir = filepath.Join(filepath.Dir(execPath), "configs")
		}
	}

	log.Printf("Loading configuration from: %s", configDir)

	// Create configuration loader
	loader := config.NewLoader(configDir)

	// Load configuration
	appConfig, err := loader.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Add configuration validation at startup
	if result := appConfig.Validate(); !result.Valid {
		return nil, fmt.Errorf("configuration validation failed: %v", result.Errors)
	}

	return appConfig, nil
}

// logConfigurationStatus logs the configuration status during startup
func logConfigurationStatus(appConfig *config.AppConfig) {
	log.Println("=== Configuration Status ===")

	// Context configuration
	log.Printf("Context Configuration:")
	log.Printf("  Cleanup Interval: %v", appConfig.Context.CleanupInterval)
	log.Printf("  Session Timeout: %v", appConfig.Context.SessionTimeout)
	log.Printf("  Max Sessions: %d", appConfig.Context.MaxSessions)
	log.Printf("  Max Memory: %d MB", appConfig.Context.MaxMemoryMB)
	log.Printf("  Persistence Enabled: %t", appConfig.Context.EnablePersistence)
	log.Printf("  Persistence Path: %s", appConfig.Context.PersistencePath)
	log.Printf("  Persistence Format: %s", appConfig.Context.PersistenceFormat)
	log.Printf("  Persistence Interval: %v", appConfig.Context.PersistenceInterval)
	log.Printf("  Async Persistence: %t", appConfig.Context.EnableAsyncPersistence)

	// Server configuration
	log.Printf("Server Configuration:")
	log.Printf("  Host: %s", appConfig.Server.Host)
	log.Printf("  Port: %s", appConfig.Server.Port)
	log.Printf("  Read Timeout: %v", appConfig.Server.ReadTimeout)
	log.Printf("  Write Timeout: %v", appConfig.Server.WriteTimeout)
	log.Printf("  Idle Timeout: %v", appConfig.Server.IdleTimeout)
	log.Printf("  Shutdown Timeout: %v", appConfig.Server.ShutdownTimeout)
	log.Printf("  Max Request Size: %d bytes", appConfig.Server.MaxRequestSize)

	// Models configuration
	log.Printf("Models Configuration:")
	log.Printf("  Default Provider: %s", appConfig.Models.DefaultProvider)
	log.Printf("  Available Providers: %d", len(appConfig.Models.Providers))
	for name, provider := range appConfig.Models.Providers {
		log.Printf("    %s: %s (%s)", name, provider.ModelName, provider.Provider)
	}

	// Prompts configuration
	log.Printf("Prompts Configuration:")
	log.Printf("  System Prompts: %d", len(appConfig.Prompts.SystemPrompts))
	log.Printf("  Examples: %d", len(appConfig.Prompts.Examples))
	log.Printf("  Max Input Length: %d", appConfig.Rules.PromptValidation.MaxInputLength)
	log.Printf("  Max Output Length: %d", appConfig.Rules.PromptValidation.MaxOutputLength)
	log.Printf("  Required Fields: %v", appConfig.Rules.PromptValidation.RequiredFields)
	// Which system prompt keys are present
	keys := []string{"base", "claude_specific", "openai_specific", "generic_specific"}
	for _, k := range keys {
		if _, ok := appConfig.Prompts.SystemPrompts[k]; ok {
			log.Printf("  System Prompt available: %s", k)
		}
	}
	// Active templates presence
	if t := appConfig.Prompts.Formats.Claude.Template; t != "" {
		log.Printf("  Formatter template active: claude")
	}
	if t := appConfig.Prompts.Formats.OpenAI.Template; t != "" {
		log.Printf("  Formatter template active: openai")
	}
	if t := appConfig.Prompts.Formats.Generic.Template; t != "" {
		log.Printf("  Formatter template active: generic")
	}

	// Rules configuration
	log.Printf("Rules Configuration:")
	log.Printf("  Allowed Log Sources: %d", len(appConfig.Rules.SafetyRules.AllowedLogSources))
	log.Printf("  Allowed Verbs: %d", len(appConfig.Rules.SafetyRules.AllowedVerbs))
	log.Printf("  Allowed Resources: %d", len(appConfig.Rules.SafetyRules.AllowedResources))
	log.Printf("  Forbidden Patterns: %d", len(appConfig.Rules.SafetyRules.ForbiddenPatterns))
	log.Printf("  Max Query Length: %d", appConfig.Rules.Sanitization.MaxQueryLength)
	log.Printf("  Max Days Back: %d", appConfig.Rules.SafetyRules.TimeframeLimits.MaxDaysBack)
	log.Printf("  Default Query Limit: %d", appConfig.Rules.SafetyRules.TimeframeLimits.DefaultLimit)
	log.Printf("  Max Query Limit: %d", appConfig.Rules.SafetyRules.TimeframeLimits.MaxLimit)
	log.Printf("  Business Hours Presets: %v", appConfig.Rules.BusinessHours.AllowedPresets)
	log.Printf("  Prompt Validation - Max Input: %d", appConfig.Rules.PromptValidation.MaxInputLength)
	log.Printf("  Prompt Validation - Max Output: %d", appConfig.Rules.PromptValidation.MaxOutputLength)
	log.Printf("  Prompt Validation - Required Fields: %v", appConfig.Rules.PromptValidation.RequiredFields)
	log.Printf("  Prompt Validation - Forbidden Words: %d", len(appConfig.Rules.PromptValidation.ForbiddenWords))

	log.Println("=== Configuration Status Complete ===")
}

// initializeProcessor initializes the GenAI processor with configuration
func initializeProcessor(appConfig *config.AppConfig) (*processor.GenAIProcessor, error) {
	// Prefer config-driven initialization; fall back to default to preserve tests
	if appConfig != nil {
		p, err := processor.NewGenAIProcessorFromConfig(appConfig)
		if err == nil && p != nil {
			log.Printf("Processor initialized from config. Default provider: %s", appConfig.Models.DefaultProvider)
			return p, nil
		}
		log.Printf("Warning: config-driven processor init failed, falling back to default: %v", err)
	}

	genaiProcessor := processor.NewGenAIProcessor()
	if genaiProcessor == nil {
		return nil, fmt.Errorf("failed to create GenAI processor")
	}
	if appConfig != nil {
		log.Printf("Processor initialized with default model provider: %s", appConfig.Models.DefaultProvider)
	} else {
		log.Printf("Processor initialized with default configuration (no appConfig provided)")
	}
	return genaiProcessor, nil
}
