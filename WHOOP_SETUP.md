# 🏃‍♂️ WHOOP Integration Setup Guide

This guide will help you set up WHOOP integration for your FamBot, enabling daily morning standups with team sleep and recovery data.

## 🎯 What You'll Get

Once set up, your team will receive daily morning standup messages at 9 AM featuring:

- **Team Recovery Overview**: Average recovery scores and HRV data
- **Sleep Analysis**: Sleep scores, duration, and efficiency  
- **Individual Insights**: Each team member's latest sleep and recovery data
- **Motivational Messages**: Daily encouragement based on team performance

## 📋 Prerequisites

1. **WHOOP Developer Account**: You need to register as a WHOOP developer
2. **WHOOP Devices**: Team members need active WHOOP subscriptions
3. **Running FamBot**: Your Slack bot should already be operational

## 🔧 Step 1: Create WHOOP Application

### 1.1 Register as WHOOP Developer

1. Go to [WHOOP Developer Platform](https://developer.whoop.com)
2. Sign up with your WHOOP account (you need an active WHOOP subscription)
3. Accept the developer terms and conditions

### 1.2 Create Your Application

1. Navigate to the [Developer Dashboard](https://developer.whoop.com/dashboard)
2. Click **"Create New App"**
3. Fill in your application details:
   - **App Name**: `FamBot WHOOP Integration`
   - **Description**: `Slack bot integration for team morning standups`
   - **Redirect URI**: `http://localhost:8080/whoop/callback` (or your server URL)
   - **Scopes**: Select:
     - `read:recovery` - Access recovery data
     - `read:sleep` - Access sleep data  
     - `read:profile` - Access basic profile info
     - `read:workout` - Access workout/strain data

4. Click **"Create Application"**
5. **Save your credentials**:
   - `Client ID` (starts with a UUID format)
   - `Client Secret` (long random string)

## ⚙️ Step 2: Configure FamBot

### 2.1 Update Environment Variables

Edit your `.env` file and add the WHOOP configuration:

```env
# WHOOP API Configuration
WHOOP_CLIENT_ID=your-whoop-client-id-here
WHOOP_CLIENT_SECRET=your-whoop-client-secret-here
WHOOP_REDIRECT_URL=http://localhost:8080/whoop/callback

# Channel for morning standups
STANDUP_CHANNEL=general
```

### 2.2 Update Slack App Manifest

1. Go to your Slack app configuration at [api.slack.com/apps](https://api.slack.com/apps)
2. Navigate to **"App Manifest"**
3. Add the new slash commands to your manifest (they should already be in the `app_manifest.yml` file):
   - `/connect-whoop` - Connect WHOOP account
   - `/whoop-status` - Check individual WHOOP data
   - `/morning-report` - Generate team morning report  
   - `/disconnect-whoop` - Disconnect WHOOP account

4. Click **"Save Changes"**
5. **Reinstall your app** to the workspace to register the new commands

## 🚀 Step 3: Deploy and Test

### 3.1 Restart FamBot

```bash
# If running locally
go run cmd/main.go

# Or if using the built binary
./bin/fambot
```

You should see logs indicating:
```
WHOOP integration enabled
Starting WHOOP OAuth callback server on port 8080
```

### 3.2 Test the Integration

1. **Connect a WHOOP Account**:
   - In Slack, run `/connect-whoop`
   - Click the generated link to authorize WHOOP access
   - You should see a success page

2. **Check Status**:
   - Run `/whoop-status` to see your personal data
   - Run `/morning-report` to generate a team report

3. **Test Morning Standup**:
   - The bot will automatically send standups at 9 AM daily
   - You can manually trigger one with `/morning-report`

## 🛠️ Step 4: Production Deployment

### 4.1 Update Redirect URL

For production deployment:

1. Update your WHOOP app's redirect URI to your production URL:
   - Example: `https://your-domain.com/whoop/callback`
2. Update the `WHOOP_REDIRECT_URL` environment variable
3. Ensure your server is accessible on the internet for OAuth callbacks

### 4.2 Security Considerations

- Keep your `WHOOP_CLIENT_SECRET` secure and never commit it to version control
- Use HTTPS for production redirect URLs
- Consider using environment-specific configurations

## 👥 Step 5: Team Onboarding

### 5.1 Team Member Setup

Send this to your team members:

> 🏃‍♂️ **WHOOP Integration is Now Live!**
> 
> To see your sleep and recovery data in our daily standups:
> 1. Make sure you have an active WHOOP subscription
> 2. Run `/connect-whoop` in Slack
> 3. Click the link to connect your WHOOP account
> 4. That's it! Your data will appear in tomorrow's standup
> 
> **Commands you can use**:
> - `/whoop-status` - Check your latest data
> - `/morning-report` - See current team stats
> - `/disconnect-whoop` - Disconnect if needed

### 5.2 Privacy Notes

Let your team know:
- Only basic sleep, recovery, and strain data is accessed
- Data is used only for team standups and individual status checks
- Team members can disconnect at any time with `/disconnect-whoop`
- Raw data is not shared between team members, only formatted summaries

## 🔍 Troubleshooting

### Common Issues

**"WHOOP integration is not configured"**
- Check that `WHOOP_CLIENT_ID` and `WHOOP_CLIENT_SECRET` are set in your environment
- Restart the bot after adding the environment variables

**OAuth callback fails**
- Verify your redirect URL matches exactly (including http/https)
- Check that port 8080 is accessible (or update to your custom port)
- Ensure the OAuth server is running (check bot logs)

**No data appearing**
- WHOOP data can take a few hours to sync after workouts/sleep
- Try running `/whoop-status` to check individual data first
- Check bot logs for sync errors

**Morning standups not sending**
- Verify the `STANDUP_CHANNEL` exists and the bot has access
- Check that team members have connected their WHOOP accounts
- Bot will skip sending if no one has WHOOP data

### Debugging Commands

```bash
# Check if WHOOP service is configured
grep -i whoop /path/to/your/.env

# View bot logs for WHOOP-related errors
tail -f /path/to/your/bot.log | grep -i whoop

# Test the OAuth callback server
curl http://localhost:8080/
```

## 📊 Expected Morning Standup Message

Here's what your team will see each morning:

```
🌅 Good Morning Team! Here's how everyone's feeling today: 🌅

📊 Team Overview: 💪 Team is feeling strong!
• Average Recovery: 72%
• Average Sleep Score: 78%  
• Team Sleep Hours: 52.3h total

👥 Individual Stats:
• **John Doe:** Recovery: 🟢 75% (HRV: 42.3ms, RHR: 54bpm) • Sleep: 😊 82% (7.2h, 87% eff)
• **Jane Smith:** Recovery: 🟡 68% (HRV: 38.1ms, RHR: 58bpm) • Sleep: 😊 74% (6.8h, 82% eff)
• **Mike Johnson:** Recovery: 🟢 78% (HRV: 45.7ms, RHR: 52bpm) • Sleep: 😴 86% (8.1h, 91% eff)

🌟 Tuesday momentum building! 🚀 Good recovery levels - steady as she goes!

💡 Pro tip: Use `/whoop-status` to check individual stats or `/morning-report` for a fresh update!
```

## 🎉 You're All Set!

Your team now has WHOOP integration! The bot will:

- ✅ Send daily morning standups at 9 AM with team sleep/recovery data
- ✅ Allow individual status checks with `/whoop-status`  
- ✅ Enable manual team reports with `/morning-report`
- ✅ Provide motivational messages based on team performance
- ✅ Respect privacy with formatted summaries only

Enjoy your data-driven morning standups! 🚀

---

## 📞 Need Help?

If you run into issues:
1. Check the troubleshooting section above
2. Review bot logs for specific error messages
3. Verify all configuration steps were completed
4. Test with a single user first before rolling out to the team

Remember: WHOOP data syncs periodically, so new sleep/recovery data might take a few hours to appear in the bot.