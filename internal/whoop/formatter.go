package whoop

import (
	"fmt"
	"strings"
	"time"
)
// MessageFormatter handles formatting WHOOP data for Slack messages
type MessageFormatter struct{}

// NewMessageFormatter creates a new message formatter
func NewMessageFormatter() *MessageFormatter {
	return &MessageFormatter{}
}

// FormatMorningStandup creates a comprehensive morning standup message
func (f *MessageFormatter) FormatMorningStandup(teamData []map[string]interface{}) string {
	if len(teamData) == 0 {
		return "🌅 *Good Morning Team!* 🌅\n\nNo WHOOP data available yet. Connect your WHOOP accounts with `/connect-whoop` to see your daily stats! 🚀"
	}

	var message strings.Builder
	
	// Header
	message.WriteString("🌅 *Good Morning Team! Here's how everyone's feeling today:* 🌅\n\n")

	// Team summary stats
	teamSummary := f.calculateTeamSummary(teamData)
	message.WriteString(fmt.Sprintf("📊 *Team Overview:* %s\n", teamSummary.emoji))
	message.WriteString(fmt.Sprintf("• Average Recovery: %s\n", teamSummary.avgRecovery))
	message.WriteString(fmt.Sprintf("• Average Sleep Score: %s\n", teamSummary.avgSleep))
	message.WriteString(fmt.Sprintf("• Team Sleep Hours: %s\n\n", teamSummary.totalSleep))

	// Individual stats
	message.WriteString("👥 *Individual Stats:*\n")
	
	for _, userData := range teamData {
		userMsg := f.formatUserData(userData)
		message.WriteString(userMsg)
		message.WriteString("\n")
	}

	// Motivational footer
	footer := f.generateMotivationalFooter(teamSummary)
	message.WriteString(fmt.Sprintf("\n%s", footer))

	return message.String()
}

// TeamSummary holds aggregated team statistics
type TeamSummary struct {
	avgRecovery  string
	avgSleep     string
	totalSleep   string
	emoji        string
	recoveryNum  float64
	sleepNum     float64
}

// calculateTeamSummary computes team-wide statistics
func (f *MessageFormatter) calculateTeamSummary(teamData []map[string]interface{}) TeamSummary {
	var recoveryScores []float64
	var sleepScores []float64
	var totalSleepHours float64

	for _, userData := range teamData {
		if recoveryScore, ok := userData["recovery_score"]; ok && recoveryScore != nil {
			if score, ok := recoveryScore.(int64); ok {
				recoveryScores = append(recoveryScores, float64(score))
			}
		}
		
		if sleepScore, ok := userData["sleep_score"]; ok && sleepScore != nil {
			if score, ok := sleepScore.(int64); ok {
				sleepScores = append(sleepScores, float64(score))
			}
		}

		if durationMS, ok := userData["duration_ms"]; ok && durationMS != nil {
			if duration, ok := durationMS.(int64); ok {
				totalSleepHours += float64(duration) / (1000 * 60 * 60) // Convert ms to hours
			}
		}
	}

	// Calculate averages
	avgRecovery := 0.0
	if len(recoveryScores) > 0 {
		sum := 0.0
		for _, score := range recoveryScores {
			sum += score
		}
		avgRecovery = sum / float64(len(recoveryScores))
	}

	avgSleep := 0.0
	if len(sleepScores) > 0 {
		sum := 0.0
		for _, score := range sleepScores {
			sum += score
		}
		avgSleep = sum / float64(len(sleepScores))
	}

	// Determine team mood emoji
	emoji := f.getTeamMoodEmoji(avgRecovery, avgSleep)

	return TeamSummary{
		avgRecovery: f.formatScore(avgRecovery),
		avgSleep:    f.formatScore(avgSleep),
		totalSleep:  fmt.Sprintf("%.1fh total", totalSleepHours),
		emoji:       emoji,
		recoveryNum: avgRecovery,
		sleepNum:    avgSleep,
	}
}

// formatUserData creates a formatted string for a single user's data
func (f *MessageFormatter) formatUserData(userData map[string]interface{}) string {
	username := f.getString(userData, "username")
	realName := f.getString(userData, "real_name")
	
	// Use real name if available, otherwise username
	displayName := realName
	if displayName == "" {
		displayName = username
	}

	var parts []string

	// Recovery data
	if recoveryScore, exists := userData["recovery_score"]; exists && recoveryScore != nil {
		score := int(f.getInt64(userData, "recovery_score"))
		hrv := f.getInt64(userData, "hrv")
		rhr := f.getInt64(userData, "rhr")
		
		recoveryEmoji := f.getRecoveryEmoji(score)
		recoveryText := fmt.Sprintf("Recovery: %s %d%%", recoveryEmoji, score)
		
		if hrv > 0 && rhr > 0 {
			recoveryText += fmt.Sprintf(" (HRV: %.1fms, RHR: %dbpm)", float64(hrv), rhr)
		}
		
		parts = append(parts, recoveryText)
	}

	// Sleep data
	if sleepScore, exists := userData["sleep_score"]; exists && sleepScore != nil {
		score := int(f.getInt64(userData, "sleep_score"))
		durationMS := f.getInt64(userData, "duration_ms")
		efficiency := f.getFloat64(userData, "efficiency")
		
		sleepEmoji := f.getSleepEmoji(score)
		sleepHours := float64(durationMS) / (1000 * 60 * 60)
		sleepText := fmt.Sprintf("Sleep: %s %d%% (%.1fh", sleepEmoji, score, sleepHours)
		
		if efficiency > 0 {
			sleepText += fmt.Sprintf(", %.0f%% eff", efficiency)
		}
		sleepText += ")"
		
		parts = append(parts, sleepText)
	}

	// If no data available
	if len(parts) == 0 {
		parts = append(parts, "No recent data 📊")
	}

	// Combine all parts
	dataText := strings.Join(parts, " • ")
	
	return fmt.Sprintf("• **%s:** %s", displayName, dataText)
}

// Helper functions for data extraction
func (f *MessageFormatter) getString(data map[string]interface{}, key string) string {
	if val, exists := data[key]; exists && val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func (f *MessageFormatter) getInt64(data map[string]interface{}, key string) int64 {
	if val, exists := data[key]; exists && val != nil {
		if num, ok := val.(int64); ok {
			return num
		}
	}
	return 0
}

func (f *MessageFormatter) getFloat64(data map[string]interface{}, key string) float64 {
	if val, exists := data[key]; exists && val != nil {
		if num, ok := val.(float64); ok {
			return num
		}
	}
	return 0.0
}

// Emoji and formatting functions
func (f *MessageFormatter) getRecoveryEmoji(score int) string {
	switch {
	case score >= 75:
		return "🟢" // Green - Great recovery
	case score >= 50:
		return "🟡" // Yellow - Moderate recovery
	case score >= 25:
		return "🟠" // Orange - Low recovery
	default:
		return "🔴" // Red - Very low recovery
	}
}

func (f *MessageFormatter) getSleepEmoji(score int) string {
	switch {
	case score >= 80:
		return "😴" // Great sleep
	case score >= 60:
		return "😊" // Good sleep
	case score >= 40:
		return "😐" // Average sleep
	default:
		return "😵" // Poor sleep
	}
}

func (f *MessageFormatter) getTeamMoodEmoji(avgRecovery, avgSleep float64) string {
	avgScore := (avgRecovery + avgSleep) / 2
	
	switch {
	case avgScore >= 75:
		return "🔥 Team is ON FIRE!"
	case avgScore >= 60:
		return "💪 Team is feeling strong!"
	case avgScore >= 45:
		return "⚡ Team is ready to go!"
	case avgScore >= 30:
		return "☕ Team needs more coffee..."
	default:
		return "🆘 Team needs extra care today!"
	}
}

func (f *MessageFormatter) formatScore(score float64) string {
	if score == 0 {
		return "N/A"
	}
	return fmt.Sprintf("%.0f%%", score)
}

func (f *MessageFormatter) generateMotivationalFooter(summary TeamSummary) string {
	currentTime := time.Now()
	weekday := currentTime.Weekday()
	
	var dayMessage string
	switch weekday {
	case time.Monday:
		dayMessage = "Let's crush this Monday! 💪"
	case time.Tuesday:
		dayMessage = "Tuesday momentum building! 🚀"
	case time.Wednesday:
		dayMessage = "Hump day hustle! 🐪"
	case time.Thursday:
		dayMessage = "Thursday thunder! ⚡"
	case time.Friday:
		dayMessage = "Friday finisher! 🎉"
	case time.Saturday:
		dayMessage = "Saturday vibes! 🌟"
	case time.Sunday:
		dayMessage = "Sunday reset! 🧘"
	}

	// Add performance-based motivation
	performanceMsg := ""
	if summary.recoveryNum >= 70 {
		performanceMsg = "The team is well-recovered and ready for anything!"
	} else if summary.recoveryNum >= 50 {
		performanceMsg = "Good recovery levels - steady as she goes!"
	} else if summary.recoveryNum > 0 {
		performanceMsg = "Take it easy today and focus on recovery!"
	}

	footer := fmt.Sprintf("🌟 %s %s\n\n_💡 Pro tip: Use `/whoop-status` to check individual stats or `/morning-report` for a fresh update!_", 
		dayMessage, performanceMsg)

	return footer
}

// FormatUserStatus creates a detailed status message for an individual user
func (f *MessageFormatter) FormatUserStatus(userData map[string]interface{}) string {
	username := f.getString(userData, "username")
	realName := f.getString(userData, "real_name")
	
	displayName := realName
	if displayName == "" {
		displayName = username
	}

	var message strings.Builder
	message.WriteString(fmt.Sprintf("📊 *WHOOP Status for %s*\n\n", displayName))

	// Recovery section
	if recoveryScore, exists := userData["recovery_score"]; exists && recoveryScore != nil {
		score := int(f.getInt64(userData, "recovery_score"))
		hrv := f.getInt64(userData, "hrv")
		rhr := f.getInt64(userData, "rhr")
		recoveryDate := f.getString(userData, "recovery_date")
		
		recoveryEmoji := f.getRecoveryEmoji(score)
		message.WriteString(fmt.Sprintf("🔋 *Recovery:* %s %d%%\n", recoveryEmoji, score))
		if hrv > 0 {
			message.WriteString(fmt.Sprintf("   • HRV: %.1fms\n", float64(hrv)))
		}
		if rhr > 0 {
			message.WriteString(fmt.Sprintf("   • Resting HR: %d bpm\n", rhr))
		}
		if recoveryDate != "" {
			message.WriteString(fmt.Sprintf("   • Date: %s\n", recoveryDate))
		}
		message.WriteString("\n")
	}

	// Sleep section
	if sleepScore, exists := userData["sleep_score"]; exists && sleepScore != nil {
		score := int(f.getInt64(userData, "sleep_score"))
		durationMS := f.getInt64(userData, "duration_ms")
		efficiency := f.getFloat64(userData, "efficiency")
		sleepDate := f.getString(userData, "sleep_date")
		
		sleepEmoji := f.getSleepEmoji(score)
		sleepHours := float64(durationMS) / (1000 * 60 * 60)
		
		message.WriteString(fmt.Sprintf("😴 *Sleep:* %s %d%%\n", sleepEmoji, score))
		message.WriteString(fmt.Sprintf("   • Duration: %.1f hours\n", sleepHours))
		if efficiency > 0 {
			message.WriteString(fmt.Sprintf("   • Efficiency: %.0f%%\n", efficiency))
		}
		if sleepDate != "" {
			message.WriteString(fmt.Sprintf("   • Date: %s\n", sleepDate))
		}
		message.WriteString("\n")
	}

	// Strain section (if available)
	if strainScore, exists := userData["strain_score"]; exists && strainScore != nil {
		score := f.getFloat64(userData, "strain_score")
		strainDate := f.getString(userData, "strain_date")
		
		message.WriteString(fmt.Sprintf("💪 *Strain:* %.1f\n", score))
		if strainDate != "" {
			message.WriteString(fmt.Sprintf("   • Date: %s\n", strainDate))
		}
		message.WriteString("\n")
	}

	// If no data
	hasData := false
	for _, key := range []string{"recovery_score", "sleep_score", "strain_score"} {
		if _, exists := userData[key]; exists {
			hasData = true
			break
		}
	}
	
	if !hasData {
		message.WriteString("No WHOOP data available. Make sure your WHOOP account is connected!\n\n")
	}

	message.WriteString("_Use `/connect-whoop` to link your account or `/morning-report` for team stats!_")
	
	return message.String()
}