# Enhanced Task Management System - Feature Guide

This document describes the comprehensive enhancements made to the task management system, including Google Calendar integration, email notifications, enhanced CLI with fuzzy search, DAG visualization, and a typed tag system.

## 🚀 New Features

### 1. User-Aware System
- All tasks now belong to specific users
- User profiles with notification preferences
- Support for multiple users in the same system

### 2. Google Calendar Integration
- Automatic sync of active tasks to Google Calendar
- Bidirectional sync (calendar changes update tasks)
- Background sync scheduler (optional)
- Manual sync command in CLI

### 3. Email Notifications (Gmail)
- Task assignment notifications
- Task start notifications  
- Due date reminders (N days before due)
- Per-user notification preferences
- Daily reminder checker

### 4. Enhanced CLI with Positional Arguments
- Modern Cobra-based CLI with subcommands
- Positional arguments instead of flags
- Fuzzy picker for location selection (FZF)
- DAG visualization with ASCII art
- Live-updating watch mode
- Textproto-based tag editing

### 5. Typed Tag System
- Support for text, location, and time tags
- Textproto representation for easy editing
- Type-safe tag values with validation

### 6. DAG Processing and Visualization
- Topological sorting of tasks by dependencies
- ASCII art visualization showing task relationships
- Live-updating watch command

## 🛠️ Setup and Configuration

### Prerequisites

1. **Go 1.24.2 or later**
2. **Google Calendar API credentials** (optional, for calendar integration)
3. **Gmail app password** (optional, for email notifications)

### Environment Variables

For full functionality, set these environment variables:

```bash
# Google Calendar Integration
export GOOGLE_CLIENT_ID="your_google_client_id"
export GOOGLE_CLIENT_SECRET="your_google_client_secret"
export GOOGLE_REDIRECT_URL="http://localhost:8080/auth/callback"

# Email Notifications (Gmail)
export EMAIL_SMTP_HOST="smtp.gmail.com"
export EMAIL_SMTP_PORT="587"
export EMAIL_USERNAME="your_gmail@gmail.com"
export EMAIL_PASSWORD="your_gmail_app_password"
export EMAIL_FROM="your_gmail@gmail.com"

# Optional: Enable automatic calendar sync
export AUTO_SYNC_ENABLED="true"
```

### Building and Running

1. **Build the enhanced system:**
   ```bash
   make build-enhanced
   make build-enhanced-client
   ```

2. **Start the enhanced server:**
   ```bash
   make run-enhanced
   ```

3. **Use the enhanced client:**
   ```bash
   ./bin/enhanced-client.exe --user user1 --help
   ```

## 📋 CLI Usage Examples

### Basic Task Management

```bash
# Add a task for a user
./bin/enhanced-client.exe --user user1 add "Review Code" "Review the new authentication feature"

# List tasks by stage
./bin/enhanced-client.exe --user user1 list pending
./bin/enhanced-client.exe --user user1 list active

# Start a task
./bin/enhanced-client.exe --user user1 start task123

# Complete a task
./bin/enhanced-client.exe --user user1 complete task123
```

### Enhanced Features

#### Move Task with Fuzzy Location Picker
```bash
# Move task to staging - will show fuzzy picker for existing locations
./bin/enhanced-client.exe --user user1 move task123
```

#### DAG Visualization
```bash
# Show static DAG view
./bin/enhanced-client.exe --user user1 view

# Live-updating DAG view (refreshes every 5 seconds)
./bin/enhanced-client.exe --user user1 watch
```

#### Tag Management
```bash
# Set tags using textproto format
./bin/enhanced-client.exe --user user1 tags set task123
# Then enter tags like:
# "priority": {text_value: "high"}
# "location": {location_value: {address: "Office Building A"}}
# "due": {time_value: "2024-12-31T23:59:59Z"}
```

#### Calendar Integration
```bash
# Manually sync with Google Calendar
./bin/enhanced-client.exe --user user1 sync
```

#### User Management
```bash
# View user information and notification settings
./bin/enhanced-client.exe --user user1 user show
```

## 🏷️ Tag System

The enhanced tag system supports three types of tags:

### Text Tags
```
"priority": {text_value: "high"}
"status": {text_value: "urgent"}
```

### Location Tags
```
"office": {location_value: {
  latitude: 37.7749,
  longitude: -122.4194,
  address: "San Francisco Office"
}}
```

### Time Tags
```
"due": {time_value: "2024-12-31T23:59:59Z"}
"reminder": {time_value: "2024-12-30T09:00:00Z"}
```

## 📅 Calendar Integration

### Setup Process

1. **Get Google Calendar API credentials:**
   - Go to Google Cloud Console
   - Create a new project or select existing
   - Enable Google Calendar API
   - Create OAuth 2.0 credentials
   - Set redirect URI to `http://localhost:8080/auth/callback`

2. **Set environment variables** (see above)

3. **User OAuth flow:**
   - Users need to authorize the app to access their calendar
   - The server provides OAuth URLs for users to complete authorization
   - Access tokens are stored per user

### Sync Behavior

- **Automatic sync:** When `AUTO_SYNC_ENABLED=true`, syncs every 30 minutes
- **Manual sync:** Use the `sync` command in CLI
- **Bidirectional:** Changes in Google Calendar update tasks, and vice versa
- **Active tasks only:** Only tasks in "active" stage are synced to calendar

## 📧 Email Notifications

### Notification Types

1. **On Assignment:** When a task is assigned to a user
2. **On Start:** When a user starts working on a task  
3. **Due Reminders:** N days before a task is due

### Gmail Setup

1. **Enable 2-Factor Authentication** on your Gmail account
2. **Generate an App Password:**
   - Go to Google Account settings
   - Security → 2-Step Verification → App passwords
   - Generate password for "Mail"
   - Use this password in `EMAIL_PASSWORD` environment variable

### User Notification Preferences

Each user can configure which notifications they receive:

```go
// Example notification settings for a user
settings := []domain.NotificationSetting{
    {Type: domain.NotificationOnAssign, Enabled: true},
    {Type: domain.NotificationOnStart, Enabled: true},
    {Type: domain.NotificationNDaysBeforeDue, Enabled: true, DaysBefore: 3},
}
```

## 🎯 DAG Visualization

The system provides topological sorting and ASCII visualization of task dependencies:

```
📋 Task Dependency Graph (Topological Order)
==================================================

⏳ PEND [12ab34cd] Setup Database
    ↗ enables: Configure API, Write Tests

🎭 STAGE [56ef78gh] Configure API  
    ↳ depends on: Setup Database
    ↗ enables: Deploy Application

⚡ ACTIVE [9ijk01lm] Write Tests
    ↳ depends on: Setup Database

🎭 STAGE [nopq23rs] Deploy Application
    ↳ depends on: Configure API
```

### Icons Legend
- ⏳ PEND - Pending tasks
- 📥 INBOX - Inbox tasks  
- 🎭 STAGE - Staging tasks
- ⚡ ACTIVE - Active tasks
- 📦 ARCH - Archived tasks

## 🔧 Development and Testing

### Running Tests
```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage
```

### Demo Workflow
```bash
# Start the enhanced server
make run-enhanced

# In another terminal, run the demo
make demo-enhanced
```

### Development Pipeline
```bash
# Complete development workflow
make dev
```

## 🔒 Security Considerations

1. **Environment Variables:** Store sensitive credentials in environment variables, not in code
2. **OAuth Tokens:** Calendar tokens are stored per user and should be encrypted in production
3. **Email Passwords:** Use Gmail app passwords, not your main account password
4. **gRPC Security:** In production, use TLS for gRPC connections

## 🚀 Production Deployment

For production deployment:

1. **Use external databases** instead of in-memory storage
2. **Enable TLS** for gRPC connections
3. **Set up proper logging** and monitoring
4. **Use secrets management** for credentials
5. **Configure reverse proxy** for HTTP endpoints
6. **Set up backup systems** for data persistence

## 📈 Performance Considerations

- **Calendar sync:** Can be resource-intensive for many users; consider rate limiting
- **Email notifications:** Implement queuing for high-volume scenarios  
- **DAG processing:** Cached for large dependency trees
- **Fuzzy search:** Performance scales with number of existing locations

## 🐛 Troubleshooting

### Common Issues

1. **Calendar sync fails:** Check OAuth credentials and user authorization
2. **Email notifications not sent:** Verify Gmail app password and SMTP settings
3. **Fuzzy picker shows no options:** No existing locations found; will prompt for manual entry
4. **Build failures:** Run `go mod tidy` to fix dependency issues

### Debug Mode

Set environment variable for more detailed logging:
```bash
export DEBUG=true
```

This comprehensive enhancement transforms the basic task management system into a full-featured productivity tool with modern CLI UX, external integrations, and advanced visualization capabilities.
