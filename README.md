# Small RPG ADHD Monolith

A gamified habit-reward system for small groups combining Web UI and Telegram Bot functionality. Create tasks, earn coins, and spend them in the shop—all while staying motivated with your team!

## Features

- 🎯 **Task Management**: Create boolean and integer-based habit tasks with customizable rewards
- 💰 **Coin Economy**: Earn coins by completing tasks, spend them in the shop
- 👥 **Group Support**: Form teams/families, share tasks and track progress together
- 🌐 **Web Interface**: Full-featured web UI for managing everything
- 🤖 **Telegram Bot**: Complete tasks and check balance via Telegram
- 📊 **Transaction History**: Immutable ledger tracking all earnings and spending
- 🎁 **Shop System**: Create rewards and purchase them with earned coins

## Prerequisites

- **Go 1.21+** - [Install Go](https://golang.org/doc/install)
- **SQLite** - Embedded via CGo (no separate installation needed)

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd small-rpg-adhd-monolith
```

2. Install dependencies:
```bash
go mod download
```

3. Build the application:
```bash
go build -o server cmd/server/main.go
```

## Configuration

The application is configured via environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `8080` | HTTP server port |
| `SESSION_SECRET` | **Yes** (production) | `dev-secret-change-in-production` | Secret key for session encryption. **Must be changed in production!** |
| `TELEGRAM_BOT_TOKEN` | No | - | Telegram Bot API token. If not set, bot features are disabled |
| `DB_PATH` | No | `small-rpg.db` | Path to SQLite database file |

### Example Configuration

**Development:**
```bash
export PORT=8080
export SESSION_SECRET=dev-secret-change-in-production
export TELEGRAM_BOT_TOKEN=your-bot-token-here  # Optional
```

**Production:**
```bash
export PORT=8080
export SESSION_SECRET=your-strong-random-secret-here
export TELEGRAM_BOT_TOKEN=your-bot-token-here
export DB_PATH=/var/lib/small-rpg/data.db
```

### Getting a Telegram Bot Token

1. Open Telegram and search for [@BotFather](https://t.me/botfather)
2. Send `/newbot` and follow the instructions
3. Copy the token provided by BotFather
4. Set it as `TELEGRAM_BOT_TOKEN` environment variable

## Running

### Quick Start

```bash
# Run with default settings (port 8080)
go run cmd/server/main.go
```

### Production Build

```bash
# Build the binary
go build -o server cmd/server/main.go

# Run the server
./server
```

### Docker (Optional)

```bash
# Build Docker image
docker build -t small-rpg-adhd-monolith .

# Run container
docker run -p 8080:8080 \
  -e SESSION_SECRET=your-secret-here \
  -e TELEGRAM_BOT_TOKEN=your-token-here \
  -v $(pwd)/data:/data \
  small-rpg-adhd-monolith
```

## Usage

### Web Interface

1. **Access the application**:
   - Open your browser to `http://localhost:8080` (or your configured port)

2. **Create an account**:
   - Click "Register" and create your user account
   - Username and password are required

3. **Create or join a group**:
   - **Create a group**: Click "Create Group", give it a name
   - **Join existing group**: Use the invite code shared by group admin

4. **Manage tasks**:
   - Navigate to your group page
   - Create tasks with reward values:
     - **Boolean tasks**: Simple completion (e.g., "Meditate" = 10 coins)
     - **Integer tasks**: Quantity-based (e.g., "Read pages" = 1 coin per page)
   - Complete tasks to earn coins

5. **Use the shop**:
   - Create shop items with coin costs
   - Purchase rewards when you have enough coins
   - Balance is calculated across all transactions

### Telegram Bot

If `TELEGRAM_BOT_TOKEN` is configured, you can interact via Telegram:

#### Available Commands

- `/start` - Link your Telegram account to your user account
- `/balance [group]` - Check your coin balance for a group
- `/tasks [group]` - List available tasks in a group
- `/complete <task_id> [quantity]` - Complete a task
- `/groups` - List your groups
- `/help` - Show available commands

#### Linking Your Account

1. Register on the web interface first
2. Start a chat with your bot on Telegram
3. Send `/start` command
4. The bot will provide a link code
5. Go to your web profile and enter the link code
6. Your Telegram account is now linked!

#### Examples

```
/groups
→ Shows your groups with their IDs

/balance workout-group
→ Shows your balance in the "workout-group"

/tasks workout-group
→ Lists all tasks in "workout-group"

/complete 42
→ Completes boolean task with ID 42

/complete 43 5
→ Completes integer task with ID 43, quantity 5
```

## Project Structure

```
small-rpg-adhd-monolith/
├── cmd/
│   └── server/main.go              # Application entry point
├── internal/
│   ├── core/
│   │   ├── models.go               # Domain models (User, Group, Task, etc.)
│   │   └── service.go              # Business logic layer
│   ├── store/
│   │   ├── sqlite.go               # Database connection & migrations
│   │   ├── users.go                # User repository
│   │   ├── groups.go               # Group repository
│   │   ├── tasks.go                # Task & shop repository
│   │   └── transactions.go         # Transaction ledger
│   ├── web/
│   │   ├── server.go               # HTTP server setup
│   │   ├── auth.go                 # Authentication middleware
│   │   └── handlers.go             # HTTP handlers
│   └── bot/
│       └── bot.go                  # Telegram bot implementation
├── templates/
│   ├── layout.html                 # Base template
│   ├── login.html                  # Login page
│   ├── register.html               # Registration page
│   ├── dashboard.html              # User dashboard
│   └── group.html                  # Group management
├── static/
│   └── style.css                   # Stylesheet
├── go.mod                          # Go module definition
├── go.sum                          # Dependency checksums
├── ARCHITECTURE.md                 # Detailed architecture docs
└── README.md                       # This file
```

## Database Schema

The application uses SQLite with automatic migrations:

- **`users`** - User accounts with optional Telegram integration
- **`groups`** - Teams/families with unique invite codes
- **`group_members`** - Many-to-many relationship between users and groups
- **`tasks`** - Habit tasks (boolean or integer type) with reward values
- **`shop_items`** - Rewards that can be purchased with coins
- **`transactions`** - Immutable ledger of all coin movements (earnings and spending)

## Development

### Running Tests

```bash
go test ./...
```

### Building for Production

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o server cmd/server/main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o server cmd/server/main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o server.exe cmd/server/main.go
```

### Code Structure

The project follows clean architecture principles:

1. **Domain Layer** (`internal/core/`): Business logic and models
2. **Data Layer** (`internal/store/`): Database access and repositories
3. **Interface Layer** (`internal/web/`, `internal/bot/`): HTTP and Bot interfaces
4. **Entry Point** (`cmd/server/`): Application initialization

## Key Features Explained

### Task System

**Boolean Tasks**: Simple completion tracking
- Example: "Morning meditation" rewards 10 coins
- Click "Complete" → earn 10 coins

**Integer Tasks**: Quantity-based tracking
- Example: "Read pages" rewards 1 coin per page
- Enter quantity (e.g., 20 pages) → earn 20 coins

### Balance System

- Balance is calculated as the sum of all transactions
- Positive transactions: Coins earned from tasks
- Negative transactions: Coins spent in shop
- All transactions are immutable for audit trail

### Group System

- Each group has a unique invite code
- Share the code to invite members
- All members see the same tasks and shop
- Each member has their own balance per group

## Troubleshooting

### Database Locked Error

If you see "database is locked":
- Ensure only one instance is running
- Check file permissions on `small-rpg.db`

### Session Issues

If you're logged out frequently:
- Set a strong `SESSION_SECRET`
- Ensure cookies are enabled in browser

### Telegram Bot Not Responding

- Verify `TELEGRAM_BOT_TOKEN` is set correctly
- Check bot is started (look for "Telegram bot ready" in logs)
- Ensure you've linked your account via `/start`

## Security Notes

⚠️ **Important for Production**:

1. **Change SESSION_SECRET**: Use a strong random string (32+ characters)
   ```bash
   export SESSION_SECRET=$(openssl rand -base64 32)
   ```

2. **Use HTTPS**: Deploy behind a reverse proxy (nginx, Caddy) with TLS

3. **Database Backups**: Regularly backup `small-rpg.db`

4. **Bot Token Security**: Never commit your bot token to git

## License

[Your License Here]

## Support

For issues and questions:
- Check `ARCHITECTURE.md` for detailed design documentation
- Open an issue on GitHub
- Contact the maintainers

## Acknowledgments

Built with:
- Go standard library
- [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3)
- [gorilla/sessions](https://github.com/gorilla/sessions)
- [Telegram Bot API](https://core.telegram.org/bots/api)