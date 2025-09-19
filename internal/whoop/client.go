package whoop

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	BaseURL       = "https://api.prod.whoop.com/developer"
	AuthURL       = "https://api.prod.whoop.com/oauth/oauth2/auth"
	TokenURL      = "https://api.prod.whoop.com/oauth/oauth2/token"
	UserProfileURL = "/v1/user/profile/basic"
	RecoveryURL   = "/v1/recovery"
	SleepURL      = "/v1/activity/sleep"
	WorkoutURL    = "/v1/activity/workout"
)

// Client represents a WHOOP API client
type Client struct {
	httpClient   *http.Client
	clientID     string
	clientSecret string
	redirectURL  string
}

// NewClient creates a new WHOOP API client
func NewClient(clientID, clientSecret, redirectURL string) *Client {
	return &Client{
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
	}
}

// GetAuthURL returns the WHOOP OAuth authorization URL
func (c *Client) GetAuthURL(state string) string {
	params := url.Values{
		"client_id":     {c.clientID},
		"redirect_uri":  {c.redirectURL},
		"response_type": {"code"},
		"scope":         {"read:recovery read:sleep read:profile read:workout"},
		"state":         {state},
	}
	return fmt.Sprintf("%s?%s", AuthURL, params.Encode())
}

// TokenResponse represents WHOOP OAuth token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	UserID       string `json:"user_id"`
}

// ExchangeCodeForToken exchanges authorization code for access token
func (c *Client) ExchangeCodeForToken(code string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"code":          {code},
		"redirect_uri":  {c.redirectURL},
	}

	resp, err := c.httpClient.PostForm(TokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// Debug logging
	fmt.Printf("[DEBUG] Token response: AccessToken=%.10s..., ExpiresIn=%d, RefreshToken=%.10s..., UserID=%s\n", 
		tokenResp.AccessToken, tokenResp.ExpiresIn, tokenResp.RefreshToken, tokenResp.UserID)

	return &tokenResp, nil
}

// RefreshAccessToken refreshes an expired access token
func (c *Client) RefreshAccessToken(refreshToken string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"refresh_token": {refreshToken},
	}

	resp, err := c.httpClient.PostForm(TokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenResp, nil
}

// UserProfile represents basic user profile from WHOOP
type UserProfile struct {
	UserID    int64  `json:"user_id"`    // WHOOP returns user_id as a number
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// GetUserProfile fetches the user's basic profile information
func (c *Client) GetUserProfile(accessToken string) (*UserProfile, error) {
	req, err := http.NewRequest("GET", BaseURL+UserProfileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get user profile failed with status %d: %s", resp.StatusCode, string(body))
	}

	var profile UserProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to decode user profile: %w", err)
	}

	return &profile, nil
}

// RecoveryData represents WHOOP recovery data
type RecoveryData struct {
	UserID    int64     `json:"user_id"` // WHOOP returns numeric user_id
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ScoreState string   `json:"score_state"`
	Score struct {
		UserCalibrating bool    `json:"user_calibrating"`
		RecoveryScore   float64 `json:"recovery_score"` // WHOOP returns float recovery score
		HRVRmssd        float64 `json:"hrv_rmssd_milli"`
		RestingHR       float64 `json:"resting_heart_rate"` // WHOOP returns float RHR
	} `json:"score"`
}

// RecoveryResponse represents the API response for recovery data
type RecoveryResponse struct {
	Records    []RecoveryData `json:"records"`
	NextToken  string         `json:"next_token"`
}

// GetRecovery fetches recovery data for a date range
func (c *Client) GetRecovery(accessToken string, start, end time.Time) (*RecoveryResponse, error) {
	params := url.Values{
		"start": {start.Format("2006-01-02T15:04:05.000Z")},
		"end":   {end.Format("2006-01-02T15:04:05.000Z")},
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s?%s", BaseURL, RecoveryURL, params.Encode()), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get recovery data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get recovery failed with status %d: %s", resp.StatusCode, string(body))
	}

	var recoveryResp RecoveryResponse
	if err := json.NewDecoder(resp.Body).Decode(&recoveryResp); err != nil {
		return nil, fmt.Errorf("failed to decode recovery response: %w", err)
	}

	return &recoveryResp, nil
}

// SleepData represents WHOOP sleep data
type SleepData struct {
	ID        int64     `json:"id"`      // WHOOP returns numeric sleep record ID
	UserID    int64     `json:"user_id"` // WHOOP returns numeric user_id
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Score struct {
		Stage_summary struct {
			TotalInBedTimeMS    int `json:"total_in_bed_time_milli"`
			TotalAwakeTimeMS    int `json:"total_awake_time_milli"`
			TotalNoDataTimeMS   int `json:"total_no_data_time_milli"`
			TotalLightSleepMS   int `json:"total_light_sleep_time_milli"`
			TotalSlowWaveSleepMS int `json:"total_slow_wave_sleep_time_milli"`
			TotalRemSleepMS     int `json:"total_rem_sleep_time_milli"`
			SleepCycleCount     int `json:"sleep_cycle_count"`
			DisturbanceCount    int `json:"disturbance_count"`
		} `json:"stage_summary"`
		SleepNeeded struct {
			BaselineMS         int `json:"baseline_milli"`
			NeedFromSleepDebtMS int `json:"need_from_sleep_debt_milli"`
			NeedFromRecentStrainMS int `json:"need_from_recent_strain_milli"`
			NeedFromRecentNapMS    int `json:"need_from_recent_nap_milli"`
		} `json:"sleep_needed"`
		SleepEfficiencyPercentage float64 `json:"sleep_efficiency_percentage"`
		SleepConsistencyPercentage float64 `json:"sleep_consistency_percentage"`
		SleepScore                int     `json:"sleep_score"`
	} `json:"score"`
}

// SleepResponse represents the API response for sleep data
type SleepResponse struct {
	Records   []SleepData `json:"records"`
	NextToken string      `json:"next_token"`
}

// GetSleep fetches sleep data for a date range
func (c *Client) GetSleep(accessToken string, start, end time.Time) (*SleepResponse, error) {
	params := url.Values{
		"start": {start.Format("2006-01-02T15:04:05.000Z")},
		"end":   {end.Format("2006-01-02T15:04:05.000Z")},
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s?%s", BaseURL, SleepURL, params.Encode()), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get sleep data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get sleep failed with status %d: %s", resp.StatusCode, string(body))
	}

	var sleepResp SleepResponse
	if err := json.NewDecoder(resp.Body).Decode(&sleepResp); err != nil {
		return nil, fmt.Errorf("failed to decode sleep response: %w", err)
	}

	return &sleepResp, nil
}

// WorkoutData represents WHOOP workout data
type WorkoutData struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Score struct {
		Strain        float64 `json:"strain"`
		AverageHR     int     `json:"average_heart_rate"`
		MaxHR         int     `json:"max_heart_rate"`
		Kilojoule     float64 `json:"kilojoule"`
		PercentZone5  float64 `json:"percent_recorded"`
	} `json:"score"`
	Sport struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"sport"`
}

// WorkoutResponse represents the API response for workout data
type WorkoutResponse struct {
	Records   []WorkoutData `json:"records"`
	NextToken string        `json:"next_token"`
}

// GetWorkouts fetches workout/strain data for a date range
func (c *Client) GetWorkouts(accessToken string, start, end time.Time) (*WorkoutResponse, error) {
	params := url.Values{
		"start": {start.Format("2006-01-02T15:04:05.000Z")},
		"end":   {end.Format("2006-01-02T15:04:05.000Z")},
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s?%s", BaseURL, WorkoutURL, params.Encode()), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get workout data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get workout failed with status %d: %s", resp.StatusCode, string(body))
	}

	var workoutResp WorkoutResponse
	if err := json.NewDecoder(resp.Body).Decode(&workoutResp); err != nil {
		return nil, fmt.Errorf("failed to decode workout response: %w", err)
	}

	return &workoutResp, nil
}
