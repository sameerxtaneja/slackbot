package models

import (
	"time"
)

// User represents a Slack user
type User struct {
	ID       string `db:"id"`
	Username string `db:"username"`
	RealName string `db:"real_name"`
	Email    string `db:"email"`
}

// Karma represents a user's karma score
type Karma struct {
	ID        int       `db:"id"`
	UserID    string    `db:"user_id"`
	Username  string    `db:"username"`
	Score     int       `db:"score"`
	UpdatedAt time.Time `db:"updated_at"`
}

// KarmaLog represents individual karma changes
type KarmaLog struct {
	ID        int       `db:"id"`
	UserID    string    `db:"user_id"`
	GivenBy   string    `db:"given_by"`
	Reason    string    `db:"reason"`
	Change    int       `db:"change"` // +1 or -1
	Timestamp time.Time `db:"timestamp"`
	Channel   string    `db:"channel"`
}

// Birthday represents a user's birthday
type Birthday struct {
	ID       int    `db:"id"`
	UserID   string `db:"user_id"`
	Username string `db:"username"`
	Month    int    `db:"month"`    // 1-12
	Day      int    `db:"day"`      // 1-31
	Year     int    `db:"year"`     // Optional, can be 0 if not provided
	Timezone string `db:"timezone"` // Optional timezone
}

// Anniversary represents a user's work anniversary
type Anniversary struct {
	ID       int    `db:"id"`
	UserID   string `db:"user_id"`
	Username string `db:"username"`
	Month    int    `db:"month"`    // 1-12
	Day      int    `db:"day"`      // 1-31
	Year     int    `db:"year"`     // Year they started
	Timezone string `db:"timezone"` // Optional timezone
}

// SassyResponse represents pre-defined sassy responses
type SassyResponse struct {
	ID       int    `db:"id"`
	Response string `db:"response"`
	Category string `db:"category"` // e.g., "thank_you", "karma_given"
	Active   bool   `db:"active"`
}

// WHOOPConnection represents a user's WHOOP API connection
type WHOOPConnection struct {
	ID           int       `db:"id"`
	UserID       string    `db:"user_id"`       // Slack user ID
	WHOOPUserID  string    `db:"whoop_user_id"` // WHOOP user ID
	AccessToken  string    `db:"access_token"`
	RefreshToken string    `db:"refresh_token"`
	ExpiresAt    time.Time `db:"expires_at"`
	ConnectedAt  time.Time `db:"connected_at"`
	Active       bool      `db:"active"`
}

// WHOOPRecovery represents daily recovery data from WHOOP
type WHOOPRecovery struct {
	ID          int       `db:"id"`
	UserID      string    `db:"user_id"`      // Slack user ID
	WHOOPUserID string    `db:"whoop_user_id"` // WHOOP user ID
	Date        time.Time `db:"date"`          // Date of the recovery data
	Score       int       `db:"score"`         // Recovery score (0-100)
	HRV         float64   `db:"hrv"`           // Heart Rate Variability (ms)
	RHR         int       `db:"rhr"`           // Resting Heart Rate (bpm)
	CreatedAt   time.Time `db:"created_at"`
}

// WHOOPSleep represents daily sleep data from WHOOP
type WHOOPSleep struct {
	ID             int       `db:"id"`
	UserID         string    `db:"user_id"`         // Slack user ID
	WHOOPUserID    string    `db:"whoop_user_id"`   // WHOOP user ID
	Date           time.Time `db:"date"`            // Date of sleep
	DurationMS     int       `db:"duration_ms"`     // Total sleep duration in milliseconds
	Efficiency     float64   `db:"efficiency"`      // Sleep efficiency percentage (0-100)
	Score          int       `db:"score"`           // Sleep score (0-100)
	StagesDeepMS   int       `db:"stages_deep_ms"`  // Deep sleep in milliseconds
	StagesREMS     int       `db:"stages_rem_ms"`   // REM sleep in milliseconds
	StagesLightMS  int       `db:"stages_light_ms"` // Light sleep in milliseconds
	StagesWakeMS   int       `db:"stages_wake_ms"`  // Wake time in milliseconds
	CreatedAt      time.Time `db:"created_at"`
}

// WHOOPStrain represents daily strain data from WHOOP
type WHOOPStrain struct {
	ID          int       `db:"id"`
	UserID      string    `db:"user_id"`      // Slack user ID
	WHOOPUserID string    `db:"whoop_user_id"` // WHOOP user ID
	Date        time.Time `db:"date"`          // Date of strain
	Score       float64   `db:"score"`         // Strain score (0-21)
	CreatedAt   time.Time `db:"created_at"`
}
