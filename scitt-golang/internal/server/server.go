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
	Service *service.TransparencyService // Exported for CLI access
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
		Service: svc,
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
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/openapi.json", s.handleOpenAPISpec)

	// Well-known endpoints (should be at the top)
	s.mux.HandleFunc("/.well-known/scitt-configuration", s.handleSCITTConfiguration)
	s.mux.HandleFunc("/.well-known/scitt-keys", s.handleSCITTKeys)

	// SCRAPI routes
	s.mux.HandleFunc("/entries", s.handleEntries)
	s.mux.HandleFunc("/entries/", s.handleEntriesWithID)

	// Transparency Log routes
	s.mux.HandleFunc("/statements", s.handleStatements)
	s.mux.HandleFunc("/statements/", s.handleStatementsWithID)

	// C2SP tlog-tiles routes
	s.mux.HandleFunc("/checkpoint", s.handleCheckpoint)
	s.mux.HandleFunc("/tile/", s.handleTile)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	log.Printf("SCITT Transparency Service")
	log.Printf("Documentation: http://%s/", addr)

	// Wrap mux with middleware
	handler := s.loggingMiddleware(s.corsMiddleware(s.mux))

	return http.ListenAndServe(addr, handler)
}

// Close closes the server and releases resources
func (s *Server) Close() error {
	return s.Service.Close()
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

	// Validate API key
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		log.Printf("Missing Authorization header")
		http.Error(w, "Unauthorized: missing API key", http.StatusUnauthorized)
		return
	}

	// Extract Bearer token
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		log.Printf("Invalid Authorization header format")
		http.Error(w, "Unauthorized: invalid authorization format", http.StatusUnauthorized)
		return
	}

	apiKey := strings.TrimPrefix(authHeader, bearerPrefix)
	if apiKey != s.config.Server.APIKey {
		log.Printf("Invalid API key provided")
		http.Error(w, "Unauthorized: invalid API key", http.StatusUnauthorized)
		return
	}

	// Validate Content-Type (support both application/cose and application/scitt-statement+cose)
	contentType := r.Header.Get("Content-Type")
	if contentType != "" && contentType != "application/cose" && contentType != "application/scitt-statement+cose" {
		log.Printf("Invalid Content-Type: %s", contentType)
		http.Error(w, "Content-Type must be application/cose or application/scitt-statement+cose", http.StatusUnsupportedMediaType)
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

	resp, err := s.Service.RegisterStatement(req)
	if err != nil {
		log.Printf("Failed to register statement: %v", err)
		http.Error(w, fmt.Sprintf("Failed to register statement: %v", err), http.StatusBadRequest)
		return
	}

	// Return 201 with Location header pointing to receipt endpoint
	receiptURL := fmt.Sprintf("/statements/%d/receipt", resp.EntryID)
	w.Header().Set("Location", receiptURL)
	w.WriteHeader(http.StatusCreated)
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
	receipt, err := s.Service.GetReceipt(entryID)
	if err != nil {
		log.Printf("Failed to get receipt: %v", err)
		http.Error(w, "Receipt not found", http.StatusNotFound)
		return
	}

	// Return receipt as application/scitt-receipt+cose
	w.Header().Set("Content-Type", "application/scitt-receipt+cose")
	w.WriteHeader(http.StatusOK)
	w.Write(receipt)
}

// handleSCITTConfiguration handles GET /.well-known/scitt-configuration
func (s *Server) handleSCITTConfiguration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get configuration
	cfg := s.Service.GetSCITTConfiguration()

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
	keySet, err := s.Service.GetSCITTKeys()
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
		"issuer": s.config.Issuer,
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
				issuer := s.config.Server.CORS.AllowedOrigins[0]
				if issuer == "*" {
					w.Header().Set("Access-Control-Allow-Issuer", "*")
				} else {
					// Check if request issuer is in allowed list
					reqOrigin := r.Header.Get("Issuer")
					for _, allowedOrigin := range s.config.Server.CORS.AllowedOrigins {
						if reqOrigin == allowedOrigin {
							w.Header().Set("Access-Control-Allow-Issuer", reqOrigin)
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

	// Update server URL to match the actual request URL
	// This allows "Try it out" to work regardless of how the server is accessed
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	actualURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	if servers, ok := spec["servers"].([]interface{}); ok && len(servers) > 0 {
		if server, ok := servers[0].(map[string]interface{}); ok {
			server["url"] = actualURL
			server["description"] = "Current server"
		}
	}

	// Convert to JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(spec)
}

// handleCheckpoint handles GET /checkpoint (returns receipt for last entry)
func (s *Server) handleCheckpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get receipt for last entry in the log
	receipt, err := s.Service.GetLastReceipt()
	if err != nil {
		log.Printf("Failed to get checkpoint receipt: %v", err)
		http.Error(w, "Failed to get checkpoint", http.StatusInternalServerError)
		return
	}

	// Return receipt as COSE Sign1
	w.Header().Set("Content-Type", "application/scitt-receipt+cose")
	w.Header().Set("Cache-Control", "public, max-age=60") // Short-term caching (mutable)
	w.WriteHeader(http.StatusOK)
	w.Write(receipt)
}

// handleTile handles GET /tile/<L>/<N>[.p/<W>] and GET /tile/entries/<N>[.p/<W>] (C2SP tlog-tiles)
func (s *Server) handleTile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract path after /tile/
	path := strings.TrimPrefix(r.URL.Path, "/tile/")

	// Check if this is an entry tile request
	if strings.HasPrefix(path, "entries/") {
		s.handleEntryTile(w, r, path)
		return
	}

	// Parse merkle tree tile path
	parsed, err := parseTilePath(path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid tile path: %v", err), http.StatusBadRequest)
		return
	}

	// Get tile from service
	var width *int
	if parsed.IsPartial {
		width = &parsed.Width
	}

	tileData, err := s.Service.GetTile(parsed.Level, parsed.Index, width)
	if err != nil {
		log.Printf("Failed to get tile: %v", err)
		http.Error(w, "Tile not found", http.StatusNotFound)
		return
	}

	// Return tile data
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable") // Long-term caching (immutable)
	w.WriteHeader(http.StatusOK)
	w.Write(tileData)
}

// handleEntryTile handles GET /tile/entries/<N>[.p/<W>] (C2SP tlog-tiles)
func (s *Server) handleEntryTile(w http.ResponseWriter, r *http.Request, path string) {
	// Parse entry tile path
	parsed, err := parseEntryTilePath(path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid entry tile path: %v", err), http.StatusBadRequest)
		return
	}

	// Get entry tile from service
	var width *int
	if parsed.IsPartial {
		width = &parsed.Width
	}

	tileData, err := s.Service.GetEntryTile(parsed.Index, width)
	if err != nil {
		log.Printf("Failed to get entry tile: %v", err)
		http.Error(w, "Entry tile not found", http.StatusNotFound)
		return
	}

	// Check if client accepts gzip encoding
	acceptEncoding := r.Header.Get("Accept-Encoding")
	if strings.Contains(acceptEncoding, "gzip") {
		// TODO: Implement gzip compression if needed
		// For now, return uncompressed
	}

	// Return entry tile data
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable") // Long-term caching (immutable)
	w.WriteHeader(http.StatusOK)
	w.Write(tileData)
}

// parsedTilePath represents components of a parsed tile path
type parsedTilePath struct {
	Level     int
	Index     int64
	IsPartial bool
	Width     int
}

// parseTilePath parses a tile path like "0/042" or "0/x001/x234/067.p/42"
func parseTilePath(path string) (*parsedTilePath, error) {
	// Check for partial tile suffix
	var isPartial bool
	var width int
	basePath := path

	if strings.Contains(path, ".p/") {
		parts := strings.SplitN(path, ".p/", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid partial tile format")
		}
		basePath = parts[0]
		widthStr := parts[1]

		w, err := strconv.Atoi(widthStr)
		if err != nil {
			return nil, fmt.Errorf("invalid width: %w", err)
		}
		if w < 1 || w > 255 {
			return nil, fmt.Errorf("width must be between 1 and 255")
		}
		isPartial = true
		width = w
	}

	// Split level and index path
	parts := strings.SplitN(basePath, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid tile path format")
	}

	// Parse level
	level, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid level: %w", err)
	}

	// Parse index from path segments
	index, err := parseIndexPath(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid index: %w", err)
	}

	return &parsedTilePath{
		Level:     level,
		Index:     index,
		IsPartial: isPartial,
		Width:     width,
	}, nil
}

// parsedEntryTilePath represents components of a parsed entry tile path
type parsedEntryTilePath struct {
	Index     int64
	IsPartial bool
	Width     int
}

// parseEntryTilePath parses an entry tile path like "entries/042" or "entries/x001/x234/067.p/42"
func parseEntryTilePath(path string) (*parsedEntryTilePath, error) {
	// Remove "entries/" prefix
	if !strings.HasPrefix(path, "entries/") {
		return nil, fmt.Errorf("invalid entry tile path")
	}
	path = strings.TrimPrefix(path, "entries/")

	// Check for partial tile suffix
	var isPartial bool
	var width int
	indexPath := path

	if strings.Contains(path, ".p/") {
		parts := strings.SplitN(path, ".p/", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid partial entry tile format")
		}
		indexPath = parts[0]
		widthStr := parts[1]

		w, err := strconv.Atoi(widthStr)
		if err != nil {
			return nil, fmt.Errorf("invalid width: %w", err)
		}
		if w < 1 || w > 255 {
			return nil, fmt.Errorf("width must be between 1 and 255")
		}
		isPartial = true
		width = w
	}

	// Parse index from path segments
	index, err := parseIndexPath(indexPath)
	if err != nil {
		return nil, fmt.Errorf("invalid index: %w", err)
	}

	return &parsedEntryTilePath{
		Index:     index,
		IsPartial: isPartial,
		Width:     width,
	}, nil
}

// parseIndexPath parses index path segments into index number
// Handles formats like "042", "x001/000", "x001/x234/067"
func parseIndexPath(indexPath string) (int64, error) {
	segments := strings.Split(indexPath, "/")

	if len(segments) == 1 {
		// Simple 3-digit format (0-255)
		return strconv.ParseInt(segments[0], 10, 64)
	}

	// Strip x prefixes and parse values
	values := make([]int64, len(segments))
	for i, s := range segments {
		if strings.HasPrefix(s, "x") {
			val, err := strconv.ParseInt(s[1:], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid segment %s: %w", s, err)
			}
			values[i] = val
		} else {
			val, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid segment %s: %w", s, err)
			}
			values[i] = val
		}
	}

	// Try base-256 first (for indices 256-65535)
	var base256Result int64
	for _, val := range values {
		base256Result = base256Result*256 + val
	}

	// Check if this could be base-256 encoding
	allValid := true
	for _, val := range values {
		if val >= 256 {
			allValid = false
			break
		}
	}

	if allValid && base256Result < 65536 {
		return base256Result, nil
	}

	// Otherwise, it's decimal grouping (e.g., "x001/x234/067" → 1234067)
	var concatenated strings.Builder
	for _, val := range values {
		concatenated.WriteString(fmt.Sprintf("%03d", val))
	}

	return strconv.ParseInt(concatenated.String(), 10, 64)
}

// handleStatements handles both POST /statements (register) and GET /statements (query)
func (s *Server) handleStatements(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handlePostStatement(w, r)
	case http.MethodGet:
		s.handleQueryStatements(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handlePostStatement handles POST /statements (same as POST /entries)
func (s *Server) handlePostStatement(w http.ResponseWriter, r *http.Request) {
	// Validate API key
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		log.Printf("Missing Authorization header")
		http.Error(w, "Unauthorized: missing API key", http.StatusUnauthorized)
		return
	}

	// Extract Bearer token
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		log.Printf("Invalid Authorization header format")
		http.Error(w, "Unauthorized: invalid authorization format", http.StatusUnauthorized)
		return
	}

	apiKey := strings.TrimPrefix(authHeader, bearerPrefix)
	if apiKey != s.config.Server.APIKey {
		log.Printf("Invalid API key provided")
		http.Error(w, "Unauthorized: invalid API key", http.StatusUnauthorized)
		return
	}

	// Validate Content-Type
	contentType := r.Header.Get("Content-Type")
	if contentType != "" && contentType != "application/cose" && contentType != "application/scitt-statement+cose" {
		log.Printf("Invalid Content-Type: %s", contentType)
		http.Error(w, "Content-Type must be application/cose or application/scitt-statement+cose", http.StatusUnsupportedMediaType)
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

	resp, err := s.Service.RegisterStatement(req)
	if err != nil {
		log.Printf("Failed to register statement: %v", err)
		http.Error(w, fmt.Sprintf("Failed to register statement: %v", err), http.StatusBadRequest)
		return
	}

	// Return 201 with Location header pointing to receipt endpoint
	receiptURL := fmt.Sprintf("/statements/%d/receipt", resp.EntryID)
	w.Header().Set("Location", receiptURL)
	w.WriteHeader(http.StatusCreated)
}

// handleQueryStatements handles GET /statements with query parameters
func (s *Server) handleQueryStatements(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()

	// Build statement query
	var iss, sub, cty, typ *string
	if v := query.Get("iss"); v != "" {
		iss = &v
	}
	if v := query.Get("sub"); v != "" {
		sub = &v
	}
	if v := query.Get("cty"); v != "" {
		cty = &v
	}
	if v := query.Get("typ"); v != "" {
		typ = &v
	}

	// Parse limit and offset
	limit := 100 // Default limit
	if v := query.Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 {
			limit = l
			if limit > 1000 {
				limit = 1000 // Max limit
			}
		}
	}

	offset := 0
	if v := query.Get("offset"); v != "" {
		if o, err := strconv.Atoi(v); err == nil && o >= 0 {
			offset = o
		}
	}

	// Query statements
	statements, err := s.Service.QueryStatements(&service.QueryStatementsRequest{
		Iss:    iss,
		Sub:    sub,
		Cty:    cty,
		Typ:    typ,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		log.Printf("Failed to query statements: %v", err)
		http.Error(w, "Failed to query statements", http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(statements)
}

// handleStatementsWithID handles GET /statements/{entryId}/receipt
func (s *Server) handleStatementsWithID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract path after /statements/
	path := strings.TrimPrefix(r.URL.Path, "/statements/")

	// Check if this is a receipt request
	if strings.HasSuffix(path, "/receipt") {
		// Extract entry ID
		entryIDStr := strings.TrimSuffix(path, "/receipt")
		entryID, err := strconv.ParseInt(entryIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid entry ID", http.StatusBadRequest)
			return
		}

		// Get receipt
		receipt, err := s.Service.GetReceipt(entryID)
		if err != nil {
			log.Printf("Failed to get receipt: %v", err)
			http.Error(w, "Receipt not found", http.StatusNotFound)
			return
		}

		// Return receipt as application/scitt-receipt+cose
		w.Header().Set("Content-Type", "application/scitt-receipt+cose")
		w.WriteHeader(http.StatusOK)
		w.Write(receipt)
		return
	}

	// Unknown path
	http.Error(w, "Not found", http.StatusNotFound)
}
