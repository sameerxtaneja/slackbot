package whoop

import (
	"fmt"
	"log"
	"net/http"
)

// OAuthServer handles WHOOP OAuth callbacks
type OAuthServer struct {
	service *Service
	port    string
}

// NewOAuthServer creates a new OAuth callback server
func NewOAuthServer(service *Service, port string) *OAuthServer {
	return &OAuthServer{
		service: service,
		port:    port,
	}
}

// Start starts the HTTP server for OAuth callbacks
func (s *OAuthServer) Start() error {
	http.HandleFunc("/whoop/callback", s.handleCallback)
	http.HandleFunc("/", s.handleRoot)
	
	log.Printf("Starting WHOOP OAuth callback server on port %s", s.port)
	return http.ListenAndServe(":"+s.port, nil)
}

// handleCallback processes the OAuth callback
func (s *OAuthServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	// Extract authorization code and state from query parameters
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}
	
	if state == "" {
		http.Error(w, "Missing state parameter", http.StatusBadRequest)
		return
	}
	
	// Process the OAuth callback
	connection, err := s.service.HandleOAuthCallback(code, state)
	if err != nil {
		log.Printf("OAuth callback error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to connect WHOOP account: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Send success response
	successHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>WHOOP Connected!</title>
    <style>
        body { font-family: Arial, sans-serif; text-align: center; padding: 50px; }
        .success { color: #28a745; }
        .container { max-width: 500px; margin: 0 auto; }
    </style>
</head>
<body>
    <div class="container">
        <h1 class="success">🎉 WHOOP Account Connected!</h1>
        <p>Your WHOOP account has been successfully connected to FamBot.</p>
        <p>You'll now see your sleep, recovery, and strain data in morning standups!</p>
        <p><strong>You can close this window and return to Slack.</strong></p>
        <hr>
        <p><small>Use <code>/whoop-status</code> in Slack to check your stats or <code>/morning-report</code> for team data.</small></p>
    </div>
</body>
</html>`
	
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(successHTML))
	
	log.Printf("Successfully connected WHOOP account for user %s", connection.UserID)
}

// handleRoot provides basic information about the service
func (s *OAuthServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	
	infoHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>FamBot WHOOP Integration</title>
    <style>
        body { font-family: Arial, sans-serif; text-align: center; padding: 50px; }
        .container { max-width: 500px; margin: 0 auto; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🤖 FamBot WHOOP Integration</h1>
        <p>This is the OAuth callback endpoint for WHOOP integration.</p>
        <p>To connect your WHOOP account, use the <code>/connect-whoop</code> command in Slack.</p>
        <hr>
        <p><small>If you're seeing this page unexpectedly, you can safely close it.</small></p>
    </div>
</body>
</html>`
	
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(infoHTML))
}