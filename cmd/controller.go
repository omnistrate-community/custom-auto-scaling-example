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
	CurrentCapacity   int           `json:"currentCapacity"`
	Status            string        `json:"status"`
	TargetCapacity    int           `json:"targetCapacity"`
	ScalingInProgress bool          `json:"scalingInProgress"`
	LastActionTime    time.Time     `json:"lastActionTime"`
	InCooldownPeriod  bool          `json:"inCooldownPeriod"`
	CooldownRemaining time.Duration `json:"cooldownRemaining"`
	InstanceID        string        `json:"instanceId"`
	ResourceID        string        `json:"resourceId"`
	ResourceAlias     string        `json:"resourceAlias"`
}

var autoScaler *autoscaler.Autoscaler

func init() {
	// Initialize logger first
	logger.InitLogger()
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
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to encode JSON response")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		return
	}

	// Validate target capacity
	if req.TargetCapacity < 0 {
		response := ScaleResponse{
			Success: false,
			Error:   "Target capacity must be non-negative",
		}
		w.WriteHeader(http.StatusBadRequest)
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to encode JSON response")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		return
	}

	// Perform scaling operation
	ctx := context.Background()
	err := autoScaler.ScaleToTarget(ctx, req.TargetCapacity)
	if err != nil {
		logger.Warn().Err(err).Msg("Scaling failed")

		// Get current status to include in error response
		currentStatus, statusErr := autoScaler.GetStatus(ctx)

		// Check if it's an "already in progress" error
		errMsg := err.Error()
		isInProgress := false
		if len(errMsg) >= 34 && errMsg[:34] == "scaling operation already in progress" {
			isInProgress = true
			errMsg = "A scaling operation is already in progress. Please wait for it to complete."
		} else {
			errMsg = fmt.Sprintf("Scaling failed: %v", err)
		}

		if statusErr == nil {
			// Include current status information in the error response
			response := map[string]interface{}{
				"success": false,
				"error":   errMsg,
				"currentStatus": map[string]interface{}{
					"currentCapacity": currentStatus.CurrentCapacity,
					"status":          string(currentStatus.Status),
					"instanceId":      currentStatus.InstanceID,
					"resourceId":      currentStatus.ResourceID,
					"resourceAlias":   currentStatus.ResourceAlias,
				},
			}
			// Use 409 Conflict for "already in progress" errors
			if isInProgress {
				w.WriteHeader(http.StatusConflict)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				logger.Warn().Err(err).Msg("Failed to encode JSON response")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		} else {
			// If we can't get status, just return the error
			response := ScaleResponse{
				Success: false,
				Error:   errMsg,
			}
			if isInProgress {
				w.WriteHeader(http.StatusConflict)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				logger.Warn().Err(err).Msg("Failed to encode JSON response")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		return
	}

	response := ScaleResponse{
		Success: true,
		Message: fmt.Sprintf("Successfully scaled to target capacity: %d", req.TargetCapacity),
	}
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to encode JSON response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	ctx := r.Context()
	capacity, err := autoScaler.GetStatus(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get current capacity")
		response := ScaleResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to get current capacity: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to encode JSON response")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		return
	}

	response := StatusResponse{
		CurrentCapacity:   capacity.CurrentCapacity,
		Status:            string(capacity.Status),
		TargetCapacity:    capacity.TargetCapacity,
		ScalingInProgress: capacity.ScalingInProgress,
		LastActionTime:    capacity.LastActionTime,
		InCooldownPeriod:  capacity.InCooldownPeriod,
		CooldownRemaining: capacity.CooldownRemaining,
		InstanceID:        capacity.InstanceID,
		ResourceID:        capacity.ResourceID,
		ResourceAlias:     capacity.ResourceAlias,
	}

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to encode JSON response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := fmt.Fprintf(w, `{"status": "healthy", "service": "autoscaler"}`)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to write health response")
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	config := autoScaler.GetConfig()
	_, err := fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>Omnistrate Autoscaler Control Panel</title>
    <style>
        @import url('https://fonts.googleapis.com/css2?family=VT323&display=swap');
        
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'VT323', monospace;
            background: #000;
            color: #00ff00;
            min-height: 100vh;
            display: flex;
            justify-content: center;
            align-items: center;
            background-image: 
                repeating-linear-gradient(0deg, rgba(0, 255, 0, 0.03) 0px, transparent 1px, transparent 2px, rgba(0, 255, 0, 0.03) 3px),
                linear-gradient(90deg, #000, #001a00);
            animation: flicker 0.15s infinite;
        }
        
        @keyframes flicker {
            0%%, 100%% { opacity: 1; }
            50%% { opacity: 0.98; }
        }
        
        .crt {
            position: relative;
            padding: 40px;
            max-width: 900px;
            width: 100%%;
        }
        
        .crt::before {
            content: '';
            position: fixed;
            top: 0;
            left: 0;
            width: 100%%;
            height: 100%%;
            background: linear-gradient(rgba(18, 16, 16, 0) 50%%, rgba(0, 0, 0, 0.25) 50%%);
            background-size: 100%% 4px;
            pointer-events: none;
            z-index: 1000;
        }
        
        .screen {
            background: #001a00;
            border: 4px solid #00ff00;
            border-radius: 8px;
            padding: 30px;
            box-shadow: 
                0 0 40px rgba(0, 255, 0, 0.3),
                inset 0 0 100px rgba(0, 255, 0, 0.05);
            position: relative;
        }
        
        .header {
            text-align: center;
            margin-bottom: 30px;
            text-shadow: 0 0 10px #00ff00;
        }
        
        h1 {
            font-size: 48px;
            letter-spacing: 4px;
            margin-bottom: 10px;
            animation: glow 2s ease-in-out infinite;
        }
        
        @keyframes glow {
            0%%, 100%% { text-shadow: 0 0 10px #00ff00, 0 0 20px #00ff00; }
            50%% { text-shadow: 0 0 20px #00ff00, 0 0 30px #00ff00, 0 0 40px #00ff00; }
        }
        
        .subtitle {
            font-size: 20px;
            color: #00cc00;
            letter-spacing: 2px;
        }
        
        .config-box {
            background: rgba(0, 255, 0, 0.05);
            border: 2px solid #00ff00;
            padding: 20px;
            margin: 20px 0;
            font-size: 20px;
        }
        
        .config-line {
            margin: 8px 0;
            display: flex;
            justify-content: space-between;
        }
        
        .label { color: #00aa00; }
        .value { color: #00ff00; font-weight: bold; }
        
        .status-display {
            background: rgba(0, 255, 0, 0.05);
            border: 2px solid #00ff00;
            padding: 20px;
            margin: 20px 0;
            min-height: 350px;
            font-size: 20px;
        }
        
        .status-line {
            margin: 8px 0;
        }
        
        .controls {
            display: flex;
            flex-direction: column;
            gap: 15px;
            margin-top: 30px;
        }
        
        .control-group {
            display: flex;
            gap: 15px;
            align-items: stretch;
        }
        
        .control-group.scale-control {
            display: flex;
            gap: 15px;
        }
        
        .control-group.scale-control input {
            flex: 1;
            margin-bottom: 0;
        }
        
        .control-group.scale-control button {
            flex: 1;
            min-width: 200px;
        }
        
        .control-group.status-control {
            display: flex;
        }
        
        .control-group.status-control button {
            flex: 1;
        }
        
        button {
            font-family: 'VT323', monospace;
            font-size: 24px;
            padding: 18px 25px;
            background: #000;
            color: #00ff00;
            border: 3px solid #00ff00;
            cursor: pointer;
            text-transform: uppercase;
            letter-spacing: 2px;
            transition: all 0.1s;
            box-shadow: 0 0 10px rgba(0, 255, 0, 0.3);
            white-space: nowrap;
        }
        
        button:hover {
            background: #00ff00;
            color: #000;
            box-shadow: 0 0 20px rgba(0, 255, 0, 0.6);
        }
        
        button:active {
            transform: scale(0.98);
        }
        
        input[type="number"] {
            font-family: 'VT323', monospace;
            font-size: 24px;
            padding: 15px;
            background: #000;
            color: #00ff00;
            border: 2px solid #00ff00;
            margin-bottom: 10px;
            width: 100%%;
        }
        
        input[type="number"]:focus {
            outline: none;
            box-shadow: 0 0 15px rgba(0, 255, 0, 0.5);
        }
        
        .loading {
            display: none;
            text-align: center;
            font-size: 24px;
            margin: 10px 0;
            animation: blink 1s infinite;
        }
        
        @keyframes blink {
            0%%, 50%% { opacity: 1; }
            51%%, 100%% { opacity: 0; }
        }
        
        .error { color: #ff0000; text-shadow: 0 0 10px #ff0000; }
        .success { color: #00ff00; text-shadow: 0 0 10px #00ff00; }
        
        .timestamp {
            text-align: right;
            font-size: 18px;
            color: #00aa00;
            margin-top: 20px;
        }
    </style>
</head>
<body>
    <div class="crt">
        <div class="screen">
            <div class="header">
                <h1>◈ CUSTOM AUTOSCALER ◈</h1>
                <div class="subtitle">OMNISTRATE EXAMPLE</div>
            </div>
            
            <div class="config-box">
                <div class="config-line">
                    <span class="label">TARGET_RESOURCE:</span>
                    <span class="value">%s</span>
                </div>
                <div class="config-line">
                    <span class="label">COOLDOWN_DURATION:</span>
                    <span class="value">%v</span>
                </div>
            </div>
            
            <div class="status-display" id="statusDisplay">
                <div class="status-line">► SYSTEM READY</div>
                <div class="status-line">► AWAITING COMMAND...</div>
            </div>
            
            <div class="loading" id="loading">► PROCESSING...</div>
            
            <div class="controls">
                <div class="control-group scale-control">
                    <input type="number" id="targetCapacity" placeholder="Enter Target Capacity" min="0" value="1">
                    <button onclick="scaleTarget()">⚡ SCALE TARGET</button>
                </div>
                <div class="control-group status-control">
                    <button onclick="getStatus()">📊 GET STATUS</button>
                </div>
            </div>
            
            <div class="timestamp" id="timestamp"></div>
        </div>
    </div>
    
    <script>
        function updateTimestamp() {
            const now = new Date();
            document.getElementById('timestamp').textContent = 
                '◈ ' + now.toLocaleString('en-US', { 
                    year: 'numeric', month: '2-digit', day: '2-digit',
                    hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false 
                }) + ' ◈';
        }
        
        setInterval(updateTimestamp, 1000);
        updateTimestamp();
        
        function showLoading(show) {
            document.getElementById('loading').style.display = show ? 'block' : 'none';
        }
        
        function displayStatus(content, isError = false) {
            const display = document.getElementById('statusDisplay');
            display.innerHTML = content;
            if (isError) {
                display.style.color = '#ff0000';
                display.style.textShadow = '0 0 10px #ff0000';
            } else {
                display.style.color = '#00ff00';
                display.style.textShadow = '0 0 10px #00ff00';
            }
        }
        
        async function getStatus() {
            showLoading(true);
            try {
                const response = await fetch('/status');
                const data = await response.json();
                
                if (response.ok) {
                    const isFailed = data.status === 'FAILED';
                    const statusClass = isFailed ? 'error' : 'success';
                    
                    let statusDisplay = '<div class="status-line success">► STATUS RETRIEVED SUCCESSFULLY</div>' +
                        '<div class="status-line">──────────────────────────────</div>';
                    
                    // Resource Information
                    statusDisplay += '<div class="status-line ' + statusClass + '">Resource: ' + data.resourceAlias + '</div>';
                    statusDisplay += '<div class="status-line ' + statusClass + '">Instance Status: ' + data.status + '</div>';
                    statusDisplay += '<div class="status-line ' + statusClass + '">Current Capacity: ' + data.currentCapacity + '</div>';
                    
                    // Only show target capacity if scaling is in progress
                    if (data.scalingInProgress) {
                        statusDisplay += '<div class="status-line ' + statusClass + '">Target Capacity: ' + data.targetCapacity + '</div>';
                        statusDisplay += '<div class="status-line">⚙️ Scaling in progress...</div>';
                    }
                    
                    // Cooldown information
                    if (data.inCooldownPeriod) {
                        const cooldownSecs = Math.round(data.cooldownRemaining / 1000000000);
                        statusDisplay += '<div class="status-line">🕐 Cooldown Period: ' + cooldownSecs + 's remaining</div>';
                    }
                    
                    // Last action time if available
                    if (data.lastActionTime && data.lastActionTime !== '0001-01-01T00:00:00Z') {
                        const lastAction = new Date(data.lastActionTime);
                        const timeAgo = Math.round((new Date() - lastAction) / 1000);
                        let timeStr = timeAgo + 's ago';
                        if (timeAgo >= 60) {
                            timeStr = Math.round(timeAgo / 60) + 'm ago';
                        }
                        statusDisplay += '<div class="status-line">Last Action: ' + timeStr + '</div>';
                    }
                    
                    // Technical details (collapsed)
                    statusDisplay += '<div class="status-line">──────────────────────────────</div>';
                    statusDisplay += '<div class="status-line" style="opacity: 0.6;">Instance ID: ' + data.instanceId + '</div>';
                    statusDisplay += '<div class="status-line" style="opacity: 0.6;">Resource ID: ' + data.resourceId + '</div>';
                    
                    displayStatus(statusDisplay);
                } else {
                    displayStatus(
                        '<div class="status-line error">► ERROR</div>' +
                        '<div class="status-line error">' + (data.error || 'Unknown error') + '</div>',
                        true
                    );
                }
            } catch (error) {
                displayStatus(
                    '<div class="status-line error">► CONNECTION ERROR</div>' +
                    '<div class="status-line error">' + error.message + '</div>',
                    true
                );
            } finally {
                showLoading(false);
            }
        }
        
        async function scaleTarget() {
            const capacity = parseInt(document.getElementById('targetCapacity').value);
            
            if (isNaN(capacity) || capacity < 0) {
                displayStatus(
                    '<div class="status-line error">► INVALID INPUT</div>' +
                    '<div class="status-line error">Target capacity must be a non-negative number</div>',
                    true
                );
                return;
            }
            
            showLoading(true);
            try {
                const response = await fetch('/scale', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ targetCapacity: capacity })
                });
                
                const data = await response.json();
                
                if (response.ok && data.success) {
                    displayStatus(
                        '<div class="status-line success">► SCALING OPERATION SUCCESSFUL</div>' +
                        '<div class="status-line">──────────────────────────────</div>' +
                        '<div class="status-line">' + data.message + '</div>' +
                        '<div class="status-line">TARGET_CAPACITY: ' + capacity + '</div>'
                    );
                } else {
                    // If current status is included in the error response, display it first
                    let errorDisplay = '';
                    if (data.currentStatus) {
                        const isFailed = data.currentStatus.status === 'FAILED';
                        const statusClass = isFailed ? 'error' : 'success';
                        
                        errorDisplay = '<div class="status-line ' + statusClass + '">Current Status:</div>' +
                            '<div class="status-line ' + statusClass + '">Resource: ' + data.currentStatus.resourceAlias + '</div>' +
                            '<div class="status-line ' + statusClass + '">Instance Status: ' + data.currentStatus.status + '</div>' +
                            '<div class="status-line ' + statusClass + '">Current Capacity: ' + data.currentStatus.currentCapacity + '</div>' +
                            '<div class="status-line">──────────────────────────────</div>';
                    }
                    
                    // Add error message below status
                    errorDisplay += '<div class="status-line error">► SCALING FAILED</div>' +
                        '<div class="status-line error">' + (data.error || 'Unknown error') + '</div>';
                    
                    displayStatus(errorDisplay, false);
                }
            } catch (error) {
                displayStatus(
                    '<div class="status-line error">► CONNECTION ERROR</div>' +
                    '<div class="status-line error">' + error.message + '</div>',
                    true
                );
            } finally {
                showLoading(false);
            }
        }
        
        // Allow Enter key to submit
        document.getElementById('targetCapacity').addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                scaleTarget();
            }
        });
        
        // Automatically fetch status on page load
        document.addEventListener('DOMContentLoaded', function() {
            getStatus();
        });
    </script>
</body>
</html>
`, config.TargetResource, config.CooldownDuration)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to write HTML response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
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
	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize autoscaler
	var err error
	autoScaler, err = autoscaler.NewAutoscaler(ctx)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize autoscaler")
	}
	logger.Info().Msg("Autoscaler initialized successfully")

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
	cancel()

	// Shutdown server
	if err := server.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("Error during shutdown")
	}

	logger.Info().Msg("Autoscaler controller stopped")
}
