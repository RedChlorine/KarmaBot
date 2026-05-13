# KarmaBot

A powerful Telegram bot written in Go that manages user reputation systems, tracks keywords, and handles message pinning across groups.

## Overview

KarmaBot is a robust, containerized Telegram bot that brings sophisticated reputation management to Telegram groups. Built with Go and PostgreSQL, it provides real-time user tracking, dynamic keyword management, and group-wide messaging capabilities.

## ✨ Features

### Reputation System
- **Vote-based Karma**: Users gain or lose reputation through community votes
- **Keyword Triggers**: Automatic reputation changes based on configurable keywords
- **Anti-Farming**: Prevents users from voting for themselves
- **Passive Registration**: Automatically learns users when they participate in chat

### Keyword Management
- **Admin Controls**: Add and delete keywords dynamically
- **Regex Support**: Use powerful regex patterns for keyword matching
- **Positive & Negative Keywords**: Configure keywords that increase or decrease reputation
- **Real-time Updates**: Changes take effect immediately

### Message Broadcasting
- **Pin Messages**: Pin messages within individual chats
- **Broadcast to All**: Send and pin messages across all registered groups (`/pinall`)
- **Global Unpin**: Remove pins from all groups simultaneously

### Performance & Reliability
- **PostgreSQL Database**: Persistent data storage with full ACID compliance
- **In-Memory Caching**: Optimized user cache reduces database queries
- **Docker/Podman**: Fully containerized for easy deployment
- **Logging**: Critical events logged to dedicated Telegram channel

## 🛠️ Tech Stack

| Component | Version |
|-----------|---------|
| **Language** | Go 1.25+ |
| **Database** | PostgreSQL 15 |
| **Container Runtime** | Docker / Podman |
| **Telegram API** | go-telegram-bot-api/v5 |

## 📦 Dependencies

- `go-telegram-bot-api/telegram-bot-api/v5` - Telegram bot API wrapper
- `lib/pq` - PostgreSQL driver
- `joho/godotenv` - Environment configuration

## 🚀 Getting Started

### Prerequisites

- **Go 1.25+** - [Download](https://go.dev/dl/)
- **Docker** or **Podman** - Container runtime
- **Docker Compose** or **Podman Compose** - Multi-container orchestration

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/RedChlorine/KarmaBot.git
   cd KarmaBot
   ```

2. **Start the database**
   ```bash
   podman-compose up -d
   # or use docker-compose up -d
   ```

3. **Install dependencies**
   ```bash
   go mod tidy
   ```

4. **Configure environment**
   ```bash
   cp .env.example .env
   # Edit .env with your Telegram bot token and database credentials
   ```

5. **Run the bot**
   ```bash
   go run main.go
   ```

## 🔧 Configuration

KarmaBot uses environment variables for configuration. Create a `.env` file in the project root:

```env
TELEGRAM_BOT_TOKEN=your_bot_token_here
DATABASE_URL=your_DB_url_here
LOG_CHANNEL_ID=your_log_channel_id_here
ADMIN_USERNAME=your_admin_username
```

## 📚 Usage

### Bot Commands

- `/start` - Initialize the bot
- `/karma @user` - Check a user's reputation
- `/addkeyword` - Add a new keyword (admin only)
- `/delkeyword` - Remove a keyword (admin only)
- `/pin` - Pin the current message
- `/pinall` - Broadcast and pin a message to all groups
- `/unpinall` - Remove pins from all groups (admin only)

## 🐛 Troubleshooting

### Database Connection Issues
- Verify PostgreSQL container is running: `podman ps`
- Check database credentials in `.env`
- Ensure port 5432 is not in use

### Bot Not Responding
- Confirm bot token is valid
- Check Telegram channel ID in logs configuration
- Review bot logs for errors: `go run main.go`

### Cache Issues
- Clear user cache by restarting the bot
- Monitor cache performance in logs

## 📝 License

Currently Unlicenced, please contact the owner for Licencing Rights

## 👤 Author

[RedChlorine](https://github.com/RedChlorine)

## 🤝 Contributing

Contributions are welcome! Please feel free to submit pull requests or open issues for bugs and feature requests.

---

**Note**: This bot requires proper Telegram bot setup and group admin permissions to function fully.
