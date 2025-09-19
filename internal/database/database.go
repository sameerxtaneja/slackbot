package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pratikgajjar/fambot-go/internal/models"
)

// Database wraps the sql.DB connection and provides methods
type Database struct {
	db *sql.DB
}

// New creates a new database connection and initializes tables
func New(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{db: db}

	// Initialize tables
	if err := database.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	// Insert default sassy responses
	if err := database.insertDefaultSassyResponses(); err != nil {
		log.Printf("Warning: failed to insert default sassy responses: %v", err)
	}

	return database, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// createTables creates all necessary tables
func (d *Database) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			real_name TEXT,
			email TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS karma (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			username TEXT NOT NULL,
			score INTEGER DEFAULT 0,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS karma_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			given_by TEXT NOT NULL,
			reason TEXT,
			change INTEGER NOT NULL,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			channel TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS birthdays (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			username TEXT NOT NULL,
			month INTEGER NOT NULL,
			day INTEGER NOT NULL,
			year INTEGER DEFAULT 0,
			timezone TEXT DEFAULT 'UTC',
			UNIQUE(user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS anniversaries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			username TEXT NOT NULL,
			month INTEGER NOT NULL,
			day INTEGER NOT NULL,
			year INTEGER NOT NULL,
			timezone TEXT DEFAULT 'UTC',
			UNIQUE(user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS sassy_responses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			response TEXT NOT NULL,
			category TEXT NOT NULL,
			active BOOLEAN DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS whoop_connections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			whoop_user_id TEXT NOT NULL,
			access_token TEXT NOT NULL,
			refresh_token TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			connected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			active BOOLEAN DEFAULT 1,
			UNIQUE(user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS whoop_recovery (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			whoop_user_id TEXT NOT NULL,
			date DATE NOT NULL,
			score INTEGER NOT NULL,
			hrv REAL NOT NULL,
			rhr INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, date)
		)`,
		`CREATE TABLE IF NOT EXISTS whoop_sleep (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			whoop_user_id TEXT NOT NULL,
			date DATE NOT NULL,
			duration_ms INTEGER NOT NULL,
			efficiency REAL NOT NULL,
			score INTEGER NOT NULL,
			stages_deep_ms INTEGER NOT NULL,
			stages_rem_ms INTEGER NOT NULL,
			stages_light_ms INTEGER NOT NULL,
			stages_wake_ms INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, date)
		)`,
		`CREATE TABLE IF NOT EXISTS whoop_strain (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			whoop_user_id TEXT NOT NULL,
			date DATE NOT NULL,
			score REAL NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, date)
		)`,
	}

	for _, query := range queries {
		if _, err := d.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query %s: %w", query, err)
		}
	}

	return nil
}

// User operations
func (d *Database) UpsertUser(user *models.User) error {
	query := `INSERT OR REPLACE INTO users (id, username, real_name, email) VALUES (?, ?, ?, ?)`
	_, err := d.db.Exec(query, user.ID, user.Username, user.RealName, user.Email)
	return err
}

func (d *Database) GetUser(userID string) (*models.User, error) {
	query := `SELECT id, username, real_name, email FROM users WHERE id = ?`
	row := d.db.QueryRow(query, userID)

	var user models.User
	err := row.Scan(&user.ID, &user.Username, &user.RealName, &user.Email)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Karma operations
func (d *Database) GetKarma(userID string) (*models.Karma, error) {
	query := `SELECT id, user_id, username, score, updated_at FROM karma WHERE user_id = ?`
	row := d.db.QueryRow(query, userID)

	var karma models.Karma
	err := row.Scan(&karma.ID, &karma.UserID, &karma.Username, &karma.Score, &karma.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &karma, nil
}

func (d *Database) IncrementKarma(userID, username, givenBy, reason, channel string) error {
	// Start transaction
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update or insert karma
	_, err = tx.Exec(`
		INSERT INTO karma (user_id, username, score, updated_at)
		VALUES (?, ?, 1, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			score = score + 1,
			updated_at = ?`,
		userID, username, time.Now(), time.Now())
	if err != nil {
		return err
	}

	// Log the karma change
	_, err = tx.Exec(`
		INSERT INTO karma_log (user_id, given_by, reason, change, timestamp, channel)
		VALUES (?, ?, ?, 1, ?, ?)`,
		userID, givenBy, reason, time.Now(), channel)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (d *Database) GetTopKarma(limit int) ([]models.Karma, error) {
	query := `SELECT id, user_id, username, score, updated_at FROM karma ORDER BY score DESC LIMIT ?`
	rows, err := d.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var karmas []models.Karma
	for rows.Next() {
		var karma models.Karma
		err := rows.Scan(&karma.ID, &karma.UserID, &karma.Username, &karma.Score, &karma.UpdatedAt)
		if err != nil {
			return nil, err
		}
		karmas = append(karmas, karma)
	}

	return karmas, nil
}

// Birthday operations
func (d *Database) SetBirthday(birthday *models.Birthday) error {
	query := `INSERT OR REPLACE INTO birthdays (user_id, username, month, day, year, timezone) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := d.db.Exec(query, birthday.UserID, birthday.Username, birthday.Month, birthday.Day, birthday.Year, birthday.Timezone)
	return err
}

func (d *Database) GetBirthday(userID string) (*models.Birthday, error) {
	query := `SELECT id, user_id, username, month, day, year, timezone FROM birthdays WHERE user_id = ?`
	row := d.db.QueryRow(query, userID)

	var birthday models.Birthday
	err := row.Scan(&birthday.ID, &birthday.UserID, &birthday.Username, &birthday.Month, &birthday.Day, &birthday.Year, &birthday.Timezone)
	if err != nil {
		return nil, err
	}
	return &birthday, nil
}

func (d *Database) GetTodaysBirthdays() ([]models.Birthday, error) {
	now := time.Now()
	month, day := int(now.Month()), now.Day()

	query := `SELECT id, user_id, username, month, day, year, timezone FROM birthdays WHERE month = ? AND day = ?`
	rows, err := d.db.Query(query, month, day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var birthdays []models.Birthday
	for rows.Next() {
		var birthday models.Birthday
		err := rows.Scan(&birthday.ID, &birthday.UserID, &birthday.Username, &birthday.Month, &birthday.Day, &birthday.Year, &birthday.Timezone)
		if err != nil {
			return nil, err
		}
		birthdays = append(birthdays, birthday)
	}

	return birthdays, nil
}

// Anniversary operations
func (d *Database) SetAnniversary(anniversary *models.Anniversary) error {
	query := `INSERT OR REPLACE INTO anniversaries (user_id, username, month, day, year, timezone) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := d.db.Exec(query, anniversary.UserID, anniversary.Username, anniversary.Month, anniversary.Day, anniversary.Year, anniversary.Timezone)
	return err
}

func (d *Database) GetAnniversary(userID string) (*models.Anniversary, error) {
	query := `SELECT id, user_id, username, month, day, year, timezone FROM anniversaries WHERE user_id = ?`
	row := d.db.QueryRow(query, userID)

	var anniversary models.Anniversary
	err := row.Scan(&anniversary.ID, &anniversary.UserID, &anniversary.Username, &anniversary.Month, &anniversary.Day, &anniversary.Year, &anniversary.Timezone)
	if err != nil {
		return nil, err
	}
	return &anniversary, nil
}

func (d *Database) GetTodaysAnniversaries() ([]models.Anniversary, error) {
	now := time.Now()
	month, day := int(now.Month()), now.Day()

	query := `SELECT id, user_id, username, month, day, year, timezone FROM anniversaries WHERE month = ? AND day = ?`
	rows, err := d.db.Query(query, month, day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var anniversaries []models.Anniversary
	for rows.Next() {
		var anniversary models.Anniversary
		err := rows.Scan(&anniversary.ID, &anniversary.UserID, &anniversary.Username, &anniversary.Month, &anniversary.Day, &anniversary.Year, &anniversary.Timezone)
		if err != nil {
			return nil, err
		}
		anniversaries = append(anniversaries, anniversary)
	}

	return anniversaries, nil
}

// Sassy response operations
func (d *Database) GetRandomSassyResponse(category string) (*models.SassyResponse, error) {
	query := `SELECT id, response, category, active FROM sassy_responses WHERE category = ? AND active = 1 ORDER BY RANDOM() LIMIT 1`
	row := d.db.QueryRow(query, category)

	var response models.SassyResponse
	err := row.Scan(&response.ID, &response.Response, &response.Category, &response.Active)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (d *Database) insertDefaultSassyResponses() error {
	responses := []models.SassyResponse{
		{Response: "Oh, you're being polite now? How refreshing!", Category: "thank_you", Active: true},
		{Response: "Look who remembered their manners!", Category: "thank_you", Active: true},
		{Response: "Gratitude detected! Don't get used to this generosity though... 😏", Category: "thank_you", Active: true},
		{Response: "Thank you? In THIS economy?", Category: "thank_you", Active: true},
		{Response: "Well well well, someone said thank you. I'm impressed 🎭", Category: "thank_you", Active: true},
		{Response: "Karma delivered with a side of sass! You're welcome. 💅", Category: "karma_given", Active: true},
		{Response: "Another karma point hits the bank! Keep spreading those good vibes. 🏦", Category: "karma_given", Active: true},
		{Response: "Karma level up! Someone's been a good human today. 📈", Category: "karma_given", Active: true},
		{Response: "Ding! Karma deposited. Your account is looking mighty fine! 💰", Category: "karma_given", Active: true},
		{Response: "Karma inflation is real, but you earned this one! 📊", Category: "karma_given", Active: true},
		{Response: "That's nice, but how about showing some love with karma instead? Add ++ after someone's name! 😏", Category: "thank_you_no_karma", Active: true},
		{Response: "Thanks are cute and all, but karma is cuter! Try @username++ next time 💝", Category: "thank_you_no_karma", Active: true},
		{Response: "Words are wind, karma is eternal! Show your appreciation with @someone++ 🌪️✨", Category: "thank_you_no_karma", Active: true},
		{Response: "Thank you detected, but where's the karma? Don't be shy, spread those ++ vibes! 🙈", Category: "thank_you_no_karma", Active: true},
		{Response: "Appreciation noted! Now let's make it official with some karma points! @user++ 📝", Category: "thank_you_no_karma", Active: true},
		{Response: "Your gratitude is showing, but your karma game needs work! Try @someone++ 💪", Category: "thank_you_no_karma", Active: true},
		{Response: "Aww, how sweet! But you know what's sweeter? Actual karma! @username++ 🍯", Category: "thank_you_no_karma", Active: true},
		{Response: "Thank you is so yesterday. Karma is forever! Level up with @user++ 🚀", Category: "thank_you_no_karma", Active: true},
	}

	for _, response := range responses {
		// Check if response already exists
		var exists bool
		err := d.db.QueryRow("SELECT 1 FROM sassy_responses WHERE response = ?", response.Response).Scan(&exists)
		if err == sql.ErrNoRows {
			// Insert new response
			_, err = d.db.Exec("INSERT INTO sassy_responses (response, category, active) VALUES (?, ?, ?)",
				response.Response, response.Category, response.Active)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// WHOOP Connection operations
func (d *Database) UpsertWHOOPConnection(conn *models.WHOOPConnection) error {
	query := `INSERT OR REPLACE INTO whoop_connections (user_id, whoop_user_id, access_token, refresh_token, expires_at, connected_at, active) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := d.db.Exec(query, conn.UserID, conn.WHOOPUserID, conn.AccessToken, conn.RefreshToken, conn.ExpiresAt, conn.ConnectedAt, conn.Active)
	return err
}

func (d *Database) GetWHOOPConnection(userID string) (*models.WHOOPConnection, error) {
	query := `SELECT id, user_id, whoop_user_id, access_token, refresh_token, expires_at, connected_at, active FROM whoop_connections WHERE user_id = ? AND active = 1`
	row := d.db.QueryRow(query, userID)

	var conn models.WHOOPConnection
	err := row.Scan(&conn.ID, &conn.UserID, &conn.WHOOPUserID, &conn.AccessToken, &conn.RefreshToken, &conn.ExpiresAt, &conn.ConnectedAt, &conn.Active)
	if err != nil {
		return nil, err
	}
	return &conn, nil
}

func (d *Database) GetAllActiveWHOOPConnections() ([]models.WHOOPConnection, error) {
	query := `SELECT id, user_id, whoop_user_id, access_token, refresh_token, expires_at, connected_at, active FROM whoop_connections WHERE active = 1`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connections []models.WHOOPConnection
	for rows.Next() {
		var conn models.WHOOPConnection
		err := rows.Scan(&conn.ID, &conn.UserID, &conn.WHOOPUserID, &conn.AccessToken, &conn.RefreshToken, &conn.ExpiresAt, &conn.ConnectedAt, &conn.Active)
		if err != nil {
			return nil, err
		}
		connections = append(connections, conn)
	}

	return connections, nil
}

func (d *Database) DeactivateWHOOPConnection(userID string) error {
	query := `UPDATE whoop_connections SET active = 0 WHERE user_id = ?`
	_, err := d.db.Exec(query, userID)
	return err
}

// WHOOP Recovery operations
func (d *Database) UpsertWHOOPRecovery(recovery *models.WHOOPRecovery) error {
	query := `INSERT OR REPLACE INTO whoop_recovery (user_id, whoop_user_id, date, score, hrv, rhr, created_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := d.db.Exec(query, recovery.UserID, recovery.WHOOPUserID, recovery.Date, recovery.Score, recovery.HRV, recovery.RHR, recovery.CreatedAt)
	return err
}

func (d *Database) GetLatestWHOOPRecovery(userID string) (*models.WHOOPRecovery, error) {
	query := `SELECT id, user_id, whoop_user_id, date, score, hrv, rhr, created_at FROM whoop_recovery WHERE user_id = ? ORDER BY date DESC LIMIT 1`
	row := d.db.QueryRow(query, userID)

	var recovery models.WHOOPRecovery
	err := row.Scan(&recovery.ID, &recovery.UserID, &recovery.WHOOPUserID, &recovery.Date, &recovery.Score, &recovery.HRV, &recovery.RHR, &recovery.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &recovery, nil
}

func (d *Database) GetWHOOPRecoveryForDate(userID string, date time.Time) (*models.WHOOPRecovery, error) {
	dateStr := date.Format("2006-01-02")
	query := `SELECT id, user_id, whoop_user_id, date, score, hrv, rhr, created_at FROM whoop_recovery WHERE user_id = ? AND date = ?`
	row := d.db.QueryRow(query, userID, dateStr)

	var recovery models.WHOOPRecovery
	err := row.Scan(&recovery.ID, &recovery.UserID, &recovery.WHOOPUserID, &recovery.Date, &recovery.Score, &recovery.HRV, &recovery.RHR, &recovery.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &recovery, nil
}

// WHOOP Sleep operations
func (d *Database) UpsertWHOOPSleep(sleep *models.WHOOPSleep) error {
	query := `INSERT OR REPLACE INTO whoop_sleep (user_id, whoop_user_id, date, duration_ms, efficiency, score, stages_deep_ms, stages_rem_ms, stages_light_ms, stages_wake_ms, created_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := d.db.Exec(query, sleep.UserID, sleep.WHOOPUserID, sleep.Date, sleep.DurationMS, sleep.Efficiency, sleep.Score, sleep.StagesDeepMS, sleep.StagesREMS, sleep.StagesLightMS, sleep.StagesWakeMS, sleep.CreatedAt)
	return err
}

func (d *Database) GetLatestWHOOPSleep(userID string) (*models.WHOOPSleep, error) {
	query := `SELECT id, user_id, whoop_user_id, date, duration_ms, efficiency, score, stages_deep_ms, stages_rem_ms, stages_light_ms, stages_wake_ms, created_at FROM whoop_sleep WHERE user_id = ? ORDER BY date DESC LIMIT 1`
	row := d.db.QueryRow(query, userID)

	var sleep models.WHOOPSleep
	err := row.Scan(&sleep.ID, &sleep.UserID, &sleep.WHOOPUserID, &sleep.Date, &sleep.DurationMS, &sleep.Efficiency, &sleep.Score, &sleep.StagesDeepMS, &sleep.StagesREMS, &sleep.StagesLightMS, &sleep.StagesWakeMS, &sleep.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &sleep, nil
}

func (d *Database) GetWHOOPSleepForDate(userID string, date time.Time) (*models.WHOOPSleep, error) {
	dateStr := date.Format("2006-01-02")
	query := `SELECT id, user_id, whoop_user_id, date, duration_ms, efficiency, score, stages_deep_ms, stages_rem_ms, stages_light_ms, stages_wake_ms, created_at FROM whoop_sleep WHERE user_id = ? AND date = ?`
	row := d.db.QueryRow(query, userID, dateStr)

	var sleep models.WHOOPSleep
	err := row.Scan(&sleep.ID, &sleep.UserID, &sleep.WHOOPUserID, &sleep.Date, &sleep.DurationMS, &sleep.Efficiency, &sleep.Score, &sleep.StagesDeepMS, &sleep.StagesREMS, &sleep.StagesLightMS, &sleep.StagesWakeMS, &sleep.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &sleep, nil
}

// WHOOP Strain operations
func (d *Database) UpsertWHOOPStrain(strain *models.WHOOPStrain) error {
	query := `INSERT OR REPLACE INTO whoop_strain (user_id, whoop_user_id, date, score, created_at) 
			  VALUES (?, ?, ?, ?, ?)`
	_, err := d.db.Exec(query, strain.UserID, strain.WHOOPUserID, strain.Date, strain.Score, strain.CreatedAt)
	return err
}

func (d *Database) GetLatestWHOOPStrain(userID string) (*models.WHOOPStrain, error) {
	query := `SELECT id, user_id, whoop_user_id, date, score, created_at FROM whoop_strain WHERE user_id = ? ORDER BY date DESC LIMIT 1`
	row := d.db.QueryRow(query, userID)

	var strain models.WHOOPStrain
	err := row.Scan(&strain.ID, &strain.UserID, &strain.WHOOPUserID, &strain.Date, &strain.Score, &strain.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &strain, nil
}

func (d *Database) GetWHOOPStrainForDate(userID string, date time.Time) (*models.WHOOPStrain, error) {
	dateStr := date.Format("2006-01-02")
	query := `SELECT id, user_id, whoop_user_id, date, score, created_at FROM whoop_strain WHERE user_id = ? AND date = ?`
	row := d.db.QueryRow(query, userID, dateStr)

	var strain models.WHOOPStrain
	err := row.Scan(&strain.ID, &strain.UserID, &strain.WHOOPUserID, &strain.Date, &strain.Score, &strain.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &strain, nil
}

// Get all team members' latest data for morning standup
func (d *Database) GetTeamWHOOPDataForStandup() ([]map[string]interface{}, error) {
	query := `
		SELECT 
			u.id, u.username, u.real_name,
			wr.score as recovery_score, wr.hrv, wr.rhr, wr.date as recovery_date,
			ws.score as sleep_score, ws.duration_ms, ws.efficiency, ws.date as sleep_date,
			wst.score as strain_score, wst.date as strain_date
		FROM users u
		INNER JOIN whoop_connections wc ON u.id = wc.user_id AND wc.active = 1
		LEFT JOIN whoop_recovery wr ON u.id = wr.user_id AND wr.date = (
			SELECT MAX(date) FROM whoop_recovery WHERE user_id = u.id
		)
		LEFT JOIN whoop_sleep ws ON u.id = ws.user_id AND ws.date = (
			SELECT MAX(date) FROM whoop_sleep WHERE user_id = u.id
		)
		LEFT JOIN whoop_strain wst ON u.id = wst.user_id AND wst.date = (
			SELECT MAX(date) FROM whoop_strain WHERE user_id = u.id
		)`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var userID, username, realName string
		var recoveryScore, hrv, rhr sql.NullInt64
		var recoveryDate, sleepDate, strainDate sql.NullString
		var sleepScore sql.NullInt64
		var durationMS sql.NullInt64
		var efficiency, strainScore sql.NullFloat64

		err := rows.Scan(&userID, &username, &realName, &recoveryScore, &hrv, &rhr, &recoveryDate,
			&sleepScore, &durationMS, &efficiency, &sleepDate, &strainScore, &strainDate)
		if err != nil {
			return nil, err
		}

		result := map[string]interface{}{
			"user_id":   userID,
			"username":  username,
			"real_name": realName,
		}

		if recoveryScore.Valid {
			result["recovery_score"] = recoveryScore.Int64
			result["hrv"] = hrv.Int64
			result["rhr"] = rhr.Int64
			result["recovery_date"] = recoveryDate.String
		}

		if sleepScore.Valid {
			result["sleep_score"] = sleepScore.Int64
			result["duration_ms"] = durationMS.Int64
			result["efficiency"] = efficiency.Float64
			result["sleep_date"] = sleepDate.String
		}

		if strainScore.Valid {
			result["strain_score"] = strainScore.Float64
			result["strain_date"] = strainDate.String
		}

		results = append(results, result)
	}

	return results, nil
}
