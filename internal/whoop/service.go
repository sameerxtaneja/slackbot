package whoop

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/pratikgajjar/fambot-go/internal/database"
	"github.com/pratikgajjar/fambot-go/internal/models"
)

// Service handles WHOOP business logic and data synchronization
type Service struct {
	client *Client
	db     *database.Database
}

// NewService creates a new WHOOP service
func NewService(client *Client, db *database.Database) *Service {
	return &Service{
		client: client,
		db:     db,
	}
}

// GenerateState generates a random state string for OAuth
func (s *Service) GenerateState() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GetAuthURL returns the WHOOP OAuth authorization URL with state
func (s *Service) GetAuthURL(userID string) string {
	state := fmt.Sprintf("%s:%s", userID, s.GenerateState())
	return s.client.GetAuthURL(state)
}

// HandleOAuthCallback processes the OAuth callback and stores the connection
func (s *Service) HandleOAuthCallback(code, state string) (*models.WHOOPConnection, error) {
	// Extract user ID from state (format: "userID:randomState")
	if len(state) < 10 {
		return nil, fmt.Errorf("invalid state parameter")
	}
	
	var userID string
	for i, char := range state {
		if char == ':' {
			userID = state[:i]
			break
		}
	}
	if userID == "" {
		return nil, fmt.Errorf("invalid state format")
	}

	// Exchange code for tokens
	tokenResp, err := s.client.ExchangeCodeForToken(code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Get user profile to verify connection
	profile, err := s.client.GetUserProfile(tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	// Create connection record
	connection := &models.WHOOPConnection{
		UserID:       userID,
		WHOOPUserID:  fmt.Sprintf("%d", profile.UserID), // Convert numeric user ID to string
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		ConnectedAt:  time.Now(),
		Active:       true,
	}

	// Store in database
	err = s.db.UpsertWHOOPConnection(connection)
	if err != nil {
		return nil, fmt.Errorf("failed to store WHOOP connection: %w", err)
	}

	log.Printf("Successfully connected WHOOP account for user %s (WHOOP ID: %d)", userID, profile.UserID)
	return connection, nil
}

// RefreshTokenIfNeeded checks if token needs refresh and refreshes it
func (s *Service) RefreshTokenIfNeeded(connection *models.WHOOPConnection) (*models.WHOOPConnection, error) {
	// Check if token expires within next hour
	if time.Until(connection.ExpiresAt) > time.Hour {
		return connection, nil
	}

	log.Printf("Refreshing WHOOP token for user %s", connection.UserID)

	// Refresh the token
	tokenResp, err := s.client.RefreshAccessToken(connection.RefreshToken)
	if err != nil {
		// If refresh fails, deactivate the connection
		s.db.DeactivateWHOOPConnection(connection.UserID)
		return nil, fmt.Errorf("failed to refresh token for user %s: %w", connection.UserID, err)
	}

	// Update connection with new token
	connection.AccessToken = tokenResp.AccessToken
	connection.RefreshToken = tokenResp.RefreshToken
	connection.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	err = s.db.UpsertWHOOPConnection(connection)
	if err != nil {
		return nil, fmt.Errorf("failed to update WHOOP connection: %w", err)
	}

	return connection, nil
}

// SyncUserData fetches and stores the latest WHOOP data for a user
func (s *Service) SyncUserData(userID string) error {
	// Get user's WHOOP connection
	connection, err := s.db.GetWHOOPConnection(userID)
	if err != nil {
		return fmt.Errorf("no WHOOP connection found for user %s: %w", userID, err)
	}

	// Refresh token if needed
	connection, err = s.RefreshTokenIfNeeded(connection)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Sync data for the last 2 days (to handle timezone differences)
	end := time.Now()
	start := end.AddDate(0, 0, -2)

	// Sync recovery data
	if err := s.syncRecoveryData(connection, start, end); err != nil {
		log.Printf("Failed to sync recovery data for user %s: %v", userID, err)
	}

	// Sync sleep data
	if err := s.syncSleepData(connection, start, end); err != nil {
		log.Printf("Failed to sync sleep data for user %s: %v", userID, err)
	}

	return nil
}

// syncRecoveryData fetches and stores recovery data
func (s *Service) syncRecoveryData(connection *models.WHOOPConnection, start, end time.Time) error {
	recoveryResp, err := s.client.GetRecovery(connection.AccessToken, start, end)
	if err != nil {
		return fmt.Errorf("failed to get recovery data: %w", err)
	}

	for _, recovery := range recoveryResp.Records {
		// Parse the date from CreatedAt (use the recovery date)
		recoveryDate := recovery.CreatedAt.Truncate(24 * time.Hour)

		recoveryModel := &models.WHOOPRecovery{
			UserID:      connection.UserID,
			WHOOPUserID: fmt.Sprintf("%d", recovery.UserID), // Convert numeric to string
			Date:        recoveryDate,
			Score:       int(recovery.Score.RecoveryScore), // Convert float to int
			HRV:         recovery.Score.HRVRmssd,
			RHR:         int(recovery.Score.RestingHR), // Convert float to int
			CreatedAt:   time.Now(),
		}

		err := s.db.UpsertWHOOPRecovery(recoveryModel)
		if err != nil {
			log.Printf("Failed to store recovery data for user %s: %v", connection.UserID, err)
		}
	}

	return nil
}

// syncSleepData fetches and stores sleep data
func (s *Service) syncSleepData(connection *models.WHOOPConnection, start, end time.Time) error {
	sleepResp, err := s.client.GetSleep(connection.AccessToken, start, end)
	if err != nil {
		return fmt.Errorf("failed to get sleep data: %w", err)
	}

	for _, sleep := range sleepResp.Records {
		// Use the sleep end date as the date for the sleep record
		sleepDate := sleep.End.Truncate(24 * time.Hour)

		// Calculate total sleep duration from stages
		totalSleepMS := sleep.Score.Stage_summary.TotalLightSleepMS + 
					   sleep.Score.Stage_summary.TotalSlowWaveSleepMS + 
					   sleep.Score.Stage_summary.TotalRemSleepMS

		// Handle sleep score - if 0, calculate based on efficiency and duration
		sleepScore := sleep.Score.SleepScore
		if sleepScore == 0 && sleep.Score.SleepEfficiencyPercentage > 0 {
			// Estimate sleep score based on efficiency (this is a fallback)
			// WHOOP's actual algorithm is more complex, but this gives a reasonable estimate
			efficiencyFactor := sleep.Score.SleepEfficiencyPercentage / 100.0
			durationHours := float64(totalSleepMS) / (1000 * 60 * 60)
			if durationHours >= 7.5 && efficiencyFactor >= 0.85 {
				sleepScore = int(75 + (efficiencyFactor-0.85)*100) // 75-90 range
			} else if durationHours >= 6.5 && efficiencyFactor >= 0.75 {
				sleepScore = int(60 + (efficiencyFactor-0.75)*150) // 60-75 range
			} else {
				sleepScore = int(efficiencyFactor * 60) // 0-60 range
			}
			log.Printf("Sleep score was 0, estimated as %d based on %.1f%% efficiency and %.1fh duration", 
				sleepScore, sleep.Score.SleepEfficiencyPercentage, durationHours)
		}

		sleepModel := &models.WHOOPSleep{
			UserID:        connection.UserID,
			WHOOPUserID:   fmt.Sprintf("%d", sleep.UserID), // Convert numeric to string
			Date:          sleepDate,
			DurationMS:    totalSleepMS,
			Efficiency:    sleep.Score.SleepEfficiencyPercentage,
			Score:         sleepScore, // Use calculated or actual score
			StagesDeepMS:  sleep.Score.Stage_summary.TotalSlowWaveSleepMS,
			StagesREMS:    sleep.Score.Stage_summary.TotalRemSleepMS,
			StagesLightMS: sleep.Score.Stage_summary.TotalLightSleepMS,
			StagesWakeMS:  sleep.Score.Stage_summary.TotalAwakeTimeMS,
			CreatedAt:     time.Now(),
		}

		err := s.db.UpsertWHOOPSleep(sleepModel)
		if err != nil {
			log.Printf("Failed to store sleep data for user %s: %v", connection.UserID, err)
		}
	}

	return nil
}

// SyncAllUsersData syncs WHOOP data for all connected users
func (s *Service) SyncAllUsersData() error {
	connections, err := s.db.GetAllActiveWHOOPConnections()
	if err != nil {
		return fmt.Errorf("failed to get WHOOP connections: %w", err)
	}

	log.Printf("Syncing WHOOP data for %d users", len(connections))

	for _, connection := range connections {
		if err := s.SyncUserData(connection.UserID); err != nil {
			log.Printf("Failed to sync data for user %s: %v", connection.UserID, err)
			// Continue with other users
		}
	}

	log.Printf("Completed WHOOP data sync")
	return nil
}

// GetConnectionStatus returns the connection status for a user
func (s *Service) GetConnectionStatus(userID string) (*models.WHOOPConnection, error) {
	return s.db.GetWHOOPConnection(userID)
}

// DisconnectUser deactivates a user's WHOOP connection
func (s *Service) DisconnectUser(userID string) error {
	return s.db.DeactivateWHOOPConnection(userID)
}

// GetUserLatestData returns the latest WHOOP data for a user
func (s *Service) GetUserLatestData(userID string) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	// Get latest recovery data
	if recovery, err := s.db.GetLatestWHOOPRecovery(userID); err == nil {
		data["recovery"] = recovery
	}

	// Get latest sleep data  
	if sleep, err := s.db.GetLatestWHOOPSleep(userID); err == nil {
		data["sleep"] = sleep
	}

	// Get latest strain data
	if strain, err := s.db.GetLatestWHOOPStrain(userID); err == nil {
		data["strain"] = strain
	}

	return data, nil
}