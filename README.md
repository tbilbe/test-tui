# Seven Test TUI

A terminal user interface (TUI) application for testing and QA on the Seven mobile app. This tool removes the requirement to understand backend processes and data structures when testing the mobile app.

## 📥 Download & Install

### Option 1: One-Line Install (macOS/Linux)

```bash
curl -sSL https://raw.githubusercontent.com/tbilbe/test-tui/main/install.sh | bash
```

This downloads the latest release, removes macOS quarantine, and installs to `~/.local/bin`.

### Option 2: Manual Download

1. Go to the [Releases page](../../releases)
2. Download the `.tar.gz` for your platform:
   - **macOS**: `seven-test-tui_X.X.X_darwin_all.tar.gz`
   - **Linux**: `seven-test-tui_X.X.X_linux_x86_64.tar.gz`
   - **Windows**: `seven-test-tui_X.X.X_windows_x86_64.zip`

3. Extract (double-click on macOS, or `tar -xzf <file>`)

4. Open Terminal in the extracted folder and run:
   ```bash
   # macOS only - remove Apple quarantine
   xattr -d com.apple.quarantine seven-test-tui
   
   # Run it
   ./seven-test-tui
   ```

### Option 3: Build from Source

Requires Go 1.23+:
```bash
git clone https://github.com/tbilbe/test-tui.git
cd test-tui
go build -o seven-test-tui ./cmd/main.go
./seven-test-tui
```

## 🎯 Purpose

Simplifies fixture-state and read-API QA workflows by providing an intuitive interface for:
- Viewing gameweeks and fixtures
- Updating fixture states (period, time, scores)
- Managing test data without manual DynamoDB edits

> **Scope limit:** Fixture period, score, and named-scorer edits update `GameWeekFixtures` only. They do not emit player events, trigger scoring, or move league tables. Resetting fixtures to `PRE_MATCH` also does not clear player `scoredGoal` or `fixtureComplete` flags; only an Opta ingest runs that backend reset. Re-seed prefix environments periodically with the `Clone env data` GitHub workflow to remove accumulated test drift.

## 🚀 Quick Start

### Prerequisites

- **AWS CLI**: [Installation guide](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)
- **AWS SSO Access**: Access to Angstrom Seven AWS account with `seven_engineer_seven_dev` role

### 1. AWS SSO Setup

Configure AWS SSO (one-time setup):

```bash
aws configure sso
```

Use these values:
- **SSO start URL**: `https://d-99676f4f55.awsapps.com/start/#`
- **SSO region**: `eu-central-1`
- **Role**: `seven_engineer_seven_dev`
- **Default region**: `eu-west-2`
- **Profile name**: `seven_engineer_seven_dev-339713102567`

Login to activate your session:

```bash
aws sso login --profile seven_engineer_seven_dev-339713102567
export AWS_PROFILE=seven_engineer_seven_dev-339713102567
```

### 2. Set Environment Variables

```bash
export API_KEY="your-api-key"
export API_ENDPOINT="https://dev.cf.playtheseven.com" # change to your deployed prefix when needed
```

**Finding your API_KEY**: Use the same value as the mobile app's `.env` `API_KEY`.
This is sent as the `x-seven-api-key` header so requests go through the CloudFront
front door (WAF). It is required for the interactive TUI.

#### Authentication / Bearer token

The read API accepts a Seven access token: an ES256 KMS-signed JWT minted by the
Seven auth-server. It is **not** a raw bwin/Entain identity-provider token and it
is **not** a Cognito `USER_PASSWORD_AUTH` token.

Bootstrap once for each API endpoint:

```bash
seven-test-tui bootstrap
```

The command prints `{API_ENDPOINT}/oauth2/authorize`. Open it in a browser, sign
in, allow the `seven://` redirect to fail, then paste either the `code` query
parameter or the full failed deep-link URL back into the terminal. The TUI exchanges
the code at `/oauth2/token`, stores only the returned refresh token in the OS
keyring (or a `0600` file under the user config directory if no keyring is
available), and refreshes the short-lived Seven access token on startup and after a
401 read response.

The session uses a stable `device_id` of `seven-test-tui-<hostname>`. This is
intentionally distinct from a mobile device because refresh-token rotation would
otherwise invalidate that device's session.

`SEVEN_ACCESS_TOKEN` remains an escape hatch for a manually pasted **Seven** access
JWT. It bypasses stored-session refresh and is not the normal workflow:

```bash
export SEVEN_ACCESS_TOKEN="<manual Seven access JWT>"
```

**Optional overrides** (defaults are already set):
```bash
export API_ENDPOINT="https://dev.cf.playtheseven.com"
```

### 3. Run the Application

```bash
./seven-test-tui
```

**First Run** (after `seven-test-tui bootstrap`):
1. The app refreshes the stored session and opens the environment (prefix) screen
2. Enter your prefix (e.g. `se7-tomb` — casing is normalised to `SE7-` for tables)
3. The gameweek list will load
4. Use arrow keys or j/k to navigate
5. Press Enter to select a gameweek
6. Press q to quit

## 🎮 Usage

### Keyboard Shortcuts

- `↑/↓` or `j/k` - Navigate lists
- `Enter` - Select gameweek/fixture
- `Tab` - Switch between panels
- `e` - Edit selected fixture
- `r` - Refresh current view
- `q` - Quit application
- `Esc` - Close modal/go back
- `?` - Show help

### Workflow

1. **Start TUI** → the stored refresh token obtains a fresh Seven access token; use `seven-test-tui bootstrap` if no session is stored
2. **Select GameWeek** → Navigate list, press Enter
3. **View Fixtures** → See all fixtures for selected gameweek
4. **Edit Fixture** → Select fixture, press `e`
5. **Modify Fields** → Update period, time, scores, start date
6. **Save Changes** → Press Enter to save to DynamoDB
7. **Refresh** → Press `r` to see updated data

Fixture edits are for read-API state QA only; they do not exercise scoring behaviour or update player scoring data.

## 📁 Project Structure

```
seven-test-tui/
├── cmd/
│   └── main.go              # Application entry point
├── internal/
│   ├── aws/                 # AWS service clients
│   │   ├── auth.go          # Auth-server token and refresh-token storage
│   │   ├── api.go           # Backend API client
│   │   └── dynamodb.go      # DynamoDB operations
│   ├── models/              # Data models & state
│   │   ├── models.go        # GameWeek, Fixture, Team structs
│   │   └── state.go         # Application state management
│   └── ui/                  # TUI components (to be implemented)
├── pkg/
│   └── config/              # Configuration management
│       └── config.go
├── docs/
│   ├── architecture.md      # Detailed architecture docs
│   ├── backend-architecture.md
│   └── specs/
│       └── requirements-v1.md
├── go.mod
├── go.sum
└── README.md
```

## 🏗️ Architecture

See [docs/architecture.md](docs/architecture.md) for detailed architecture documentation.

**Key Components**:
- **Bubbletea**: TUI framework (Model-View-Update pattern)
- **Lipgloss**: Styling and layout
- **AWS SDK**: DynamoDB and EventBridge writes
- **Auth-server OAuth2**: Browser bootstrap and refresh-token session management

**Data Flow**:
- **Read**: API endpoints authenticated with a Seven auth-server JWT
- **Write**: Direct DynamoDB updates (using AWS credentials)

## 🚧 Current Status

**MVP Stage** - Core functionality implemented:
- ✅ Project setup and dependencies
- ✅ AWS authentication (Cognito)
- ✅ API client (gameweeks, fixtures)
- ✅ DynamoDB client (fixture updates)
- ✅ Data models with validation
- ✅ Configuration management
- 🚧 UI components (in progress)

**Future Enhancements**:
- Goal scorer management
- Customer team selection (7-player assignment)
- Real-time updates via DynamoDB Streams

## 🛠️ Development

### Build

```bash
go build -o seven-test-tui cmd/main.go
```

### Test

```bash
go test ./...
```

### Format

```bash
go fmt ./...
```

## 📝 Notes

- **Session Duration**: AWS SSO tokens last 8 hours
- **New Terminal**: Run `export AWS_PROFILE=...` in each new terminal
- **Token Expiry**: Run `aws sso login --profile ...` if you get permission errors
- **Prefix Environments**: Use personal prefix environments (e.g., `SE7-2001`) or `int-dev` for testing
- **Protected Environments**: Cannot run against `dev`, `test`, `stage`, or `prod` prefixes

## 🤝 Contributing

This project is vibe coded! Feel free to contribute improvements via Pull Requests.

## 📚 Additional Documentation

- [Architecture Overview](docs/architecture.md)
- [Backend Integration](docs/backend-architecture.md)
- [Requirements Specification](docs/specs/requirements-v1.md)