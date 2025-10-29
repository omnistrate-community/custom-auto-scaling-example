package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/omnistrate-community/custom-auto-scaling-example/internal/autoscaler"
	"github.com/omnistrate-community/custom-auto-scaling-example/internal/logger"
)

type ScaleRequest struct {
	TargetCapacity int `json:"targetCapacity"`
}

type ScaleResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

type StatusResponse struct {
	CurrentCapacity int    `json:"currentCapacity"`
	Status          string `json:"status"`
	InstanceID      string `json:"instanceId"`
	ResourceAlias   string `json:"resourceAlias"`
}

var autoScaler *autoscaler.Autoscaler

func init() {
	// Initialize logger first
	logger.InitLogger()

	ctx := context.Background()
	var err error
	autoScaler, err = autoscaler.NewAutoscaler(ctx)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize autoscaler")
	}
	logger.Info().Msg("Autoscaler initialized successfully")
}

func scaleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Parse request body
	var req ScaleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := ScaleResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid JSON: %v", err),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate target capacity
	if req.TargetCapacity < 0 {
		response := ScaleResponse{
			Success: false,
			Error:   "Target capacity must be non-negative",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Perform scaling operation
	ctx := r.Context()
	err := autoScaler.ScaleToTarget(ctx, req.TargetCapacity)
	if err != nil {
		logger.Error().Err(err).Msg("Scaling failed")
		response := ScaleResponse{
			Success: false,
			Error:   fmt.Sprintf("Scaling failed: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ScaleResponse{
		Success: true,
		Message: fmt.Sprintf("Successfully scaled to target capacity: %d", req.TargetCapacity),
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	ctx := r.Context()
	capacity, err := autoScaler.GetCurrentCapacity(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get current capacity")
		response := ScaleResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to get current capacity: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := StatusResponse{
		CurrentCapacity: capacity.CurrentCapacity,
		Status:          string(capacity.Status),
		InstanceID:      capacity.InstanceID,
		ResourceAlias:   capacity.ResourceAlias,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "healthy", "service": "autoscaler"}`)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	config := autoScaler.GetConfig()
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>Omnistrate Autoscaler</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .container { max-width: 800px; margin: 0 auto; }
        .config { background: #f5f5f5; padding: 20px; border-radius: 5px; margin-bottom: 30px; }
        .endpoint { background: #e8f4fd; padding: 15px; border-radius: 5px; margin: 10px 0; }
        code { background: #f0f0f0; padding: 2px 4px; border-radius: 3px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Omnistrate Autoscaler Service</h1>
        
        <div class="config">
            <h2>Configuration</h2>
            <p><strong>Target Resource:</strong> %s</p>
            <p><strong>Cooldown Duration:</strong> %v</p>
        </div>

        <h2>Available Endpoints</h2>
        
        <div class="endpoint">
            <h3>POST /scale</h3>
            <p>Scale the resource to target capacity</p>
            <p><strong>Request body:</strong> <code>{"targetCapacity": 5}</code></p>
        </div>
        
        <div class="endpoint">
            <h3>GET /status</h3>
            <p>Get current capacity and status</p>
        </div>
        
        <div class="endpoint">
            <h3>GET /health</h3>
            <p>Health check endpoint</p>
        </div>
    </div>
</body>
</html>
`, config.TargetResource, config.CooldownDuration)
}

/**
 * Autoscaler controller main function
 *
 * The controller reads configuration from environment variables:
 * - AUTOSCALER_COOLDOWN: Cooldown period in seconds (default: 300)
 * - AUTOSCALER_TARGET_RESOURCE: Resource alias to scale
 *
 * It exposes HTTP endpoints:
 * - POST /scale: Scale to target capacity
 * - GET /status: Get current capacity and status
 * - GET /health: Health check
 *
 * The autoscaler will:
 * 1. Get current capacity using omnistrate_api
 * 2. Wait for instance to be ACTIVE if not already
 * 3. Respect cooldown period between scaling operations
 * 4. Add or remove capacity to match target
 */
func main() {
	// Setup HTTP routes
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/scale", scaleHandler)
	http.HandleFunc("/status", statusHandler)
	http.HandleFunc("/health", healthHandler)

	// Setup graceful shutdown
	chExit := make(chan os.Signal, 1)
	signal.Notify(chExit, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP server in goroutine
	port := "3000"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info().Str("port", port).Msg("Starting autoscaler controller")
		logger.Info().Msg("Environment variables required:")
		logger.Info().Msg("  - AUTOSCALER_TARGET_RESOURCE: Resource alias to scale")
		logger.Info().Msg("  - AUTOSCALER_COOLDOWN: Cooldown period in seconds (optional)")
		logger.Info().Msg("  - AUTOSCALER_STEPS: Number of steps for scaling (optional)")
		logger.Info().Msg("")
		logger.Info().Msg("Available endpoints:")
		logger.Info().Msg("  POST /scale - Scale to target capacity")
		logger.Info().Msg("  GET /status - Get current status")
		logger.Info().Msg("  GET /health - Health check")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	// Wait for shutdown signal
	<-chExit
	logger.Info().Msg("Shutting down gracefully...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown server
	if err := server.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("Error during shutdown")
	}

	logger.Info().Msg("Autoscaler controller stopped")
}
