package server

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/config"
	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/service"
	"gopkg.in/yaml.v3"
)

//go:embed openapi.yaml
var openapiSpec string

// Server represents the HTTP server
type Server struct {
	config  *config.Config
	service *service.TransparencyService
	mux     *http.ServeMux
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config) (*Server, error) {
	// Create transparency service
	svc, err := service.NewTransparencyService(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create transparency service: %w", err)
	}

	server := &Server{
		config:  cfg,
		service: svc,
		mux:     http.NewServeMux(),
	}

	// Register routes
	server.registerRoutes()

	return server, nil
}

// registerRoutes registers all HTTP routes
func (s *Server) registerRoutes() {
	// API Documentation
	s.mux.HandleFunc("/", s.handleSwaggerUI)
	s.mux.HandleFunc("/openapi.json", s.handleOpenAPISpec)

	// Well-known endpoints (should be at the top)
	s.mux.HandleFunc("/.well-known/scitt-configuration", s.handleSCITTConfiguration)
	s.mux.HandleFunc("/.well-known/scitt-keys", s.handleSCITTKeys)

	// SCRAPI routes
	s.mux.HandleFunc("/entries", s.handleEntries)
	s.mux.HandleFunc("/entries/", s.handleEntriesWithID)
	s.mux.HandleFunc("/checkpoint", s.handleCheckpoint)

	// Health check
	s.mux.HandleFunc("/health", s.handleHealth)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	log.Printf("Starting SCITT Transparency Service on %s", addr)
	log.Printf("Documentation: http://%s/", addr)

	// Wrap mux with middleware
	handler := s.loggingMiddleware(s.corsMiddleware(s.mux))

	return http.ListenAndServe(addr, handler)
}

// Close closes the server and releases resources
func (s *Server) Close() error {
	return s.service.Close()
}

// Handler returns the HTTP handler for testing
func (s *Server) Handler() http.Handler {
	return s.loggingMiddleware(s.corsMiddleware(s.mux))
}

// handleEntries handles POST /entries (register statement)
func (s *Server) handleEntries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Register statement
	req := &service.RegisterStatementRequest{
		Statement: body,
	}

	resp, err := s.service.RegisterStatement(req)
	if err != nil {
		log.Printf("Failed to register statement: %v", err)
		http.Error(w, fmt.Sprintf("Failed to register statement: %v", err), http.StatusBadRequest)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// handleEntriesWithID handles GET /entries/{entryId} (get receipt)
func (s *Server) handleEntriesWithID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract entry ID from path
	path := strings.TrimPrefix(r.URL.Path, "/entries/")
	entryID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		http.Error(w, "Invalid entry ID", http.StatusBadRequest)
		return
	}

	// Get receipt
	receipt, err := s.service.GetReceipt(entryID)
	if err != nil {
		log.Printf("Failed to get receipt: %v", err)
		http.Error(w, "Receipt not found", http.StatusNotFound)
		return
	}

	// Return receipt
	w.Header().Set("Content-Type", "application/cbor")
	w.WriteHeader(http.StatusOK)
	w.Write(receipt)
}

// handleCheckpoint handles GET /checkpoint
func (s *Server) handleCheckpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get checkpoint
	checkpoint, err := s.service.GetCheckpoint()
	if err != nil {
		log.Printf("Failed to get checkpoint: %v", err)
		http.Error(w, "Failed to get checkpoint", http.StatusInternalServerError)
		return
	}

	// Return checkpoint
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(checkpoint))
}

// handleSCITTConfiguration handles GET /.well-known/scitt-configuration
func (s *Server) handleSCITTConfiguration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get configuration
	cfg := s.service.GetSCITTConfiguration()

	// Return configuration
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(cfg)
}

// handleSCITTKeys handles GET /.well-known/scitt-keys
func (s *Server) handleSCITTKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get keys as COSE Key Set in CBOR format
	keySet, err := s.service.GetSCITTKeys()
	if err != nil {
		log.Printf("Failed to get SCITT keys: %v", err)
		http.Error(w, "Failed to get keys", http.StatusInternalServerError)
		return
	}

	// Return COSE Key Set as CBOR (per SCRAPI specification)
	w.Header().Set("Content-Type", "application/cbor")
	w.WriteHeader(http.StatusOK)
	w.Write(keySet)
}

// handleHealth handles GET /health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := map[string]interface{}{
		"status": "healthy",
		"origin": s.config.Origin,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(health)
}

// loggingMiddleware logs all HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware adds CORS headers if configured
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.config.Server.CORS.Enabled {
			// Set CORS headers
			if len(s.config.Server.CORS.AllowedOrigins) > 0 {
				origin := s.config.Server.CORS.AllowedOrigins[0]
				if origin == "*" {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else {
					// Check if request origin is in allowed list
					reqOrigin := r.Header.Get("Origin")
					for _, allowedOrigin := range s.config.Server.CORS.AllowedOrigins {
						if reqOrigin == allowedOrigin {
							w.Header().Set("Access-Control-Allow-Origin", reqOrigin)
							break
						}
					}
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			// Handle preflight
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// handleSwaggerUI serves the Swagger UI at the root path
func (s *Server) handleSwaggerUI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Only serve Swagger UI on exact root path
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SCITT Transparency Service API</title>
    <link rel="stylesheet" type="text/css" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.10.5/swagger-ui.css">
    <style>
        body {
            margin: 0;
            padding: 0;
        }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.10.5/swagger-ui-bundle.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.10.5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            window.ui = SwaggerUIBundle({
                url: "/openapi.json",
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout"
            });
        };
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// handleOpenAPISpec serves the OpenAPI specification in JSON format
func (s *Server) handleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse YAML to map
	var spec map[string]interface{}
	if err := yaml.Unmarshal([]byte(openapiSpec), &spec); err != nil {
		log.Printf("Failed to parse OpenAPI spec: %v", err)
		http.Error(w, "Failed to load API specification", http.StatusInternalServerError)
		return
	}

	// Update server URL to match current origin
	if servers, ok := spec["servers"].([]interface{}); ok && len(servers) > 0 {
		if server, ok := servers[0].(map[string]interface{}); ok {
			server["url"] = s.config.Origin
		}
	}

	// Convert to JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(spec)
}
