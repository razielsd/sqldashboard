# SQL Dashboard
**Note:** This is experimental software provided "as-is".

A cross-platform GUI application for PostgreSQL monitoring with customizable queries and real-time data visualization.

## Why This Exists?
Tired of running the same database queries manually? SQLDashboard automates your routine data checks, giving you instant visibility without repetitive work.

## How It Works
We take multiple SQL queries to the database and store them in a config. Based on this config, the application is built, allowing us to instantly view all the results.

## Features

- 🖥️ Supports Windows, macOS and Linux
- ⚡ Automatic data refresh
- 📊 Customizable dashboard with tabs
- 🔍 Query results in sortable tables
- ⚙️ JSON configuration file
- 🔄 Configurable refresh intervals

## Installation

### Requirements

- [Go 1.20+](https://golang.org/dl/)
- PostgreSQL libraries (`libpq`)
- Git (for dependency management)

### Build from source

```bash
git clone https://github.com/razielsd/sqldashboard.git
cd sqldashboard
go build -o sqldashboard
```

#### Platform-specific builds

**Windows (GUI application):**
```bash
go build -ldflags="-H windowsgui" -o sqldashboard.exe
```

**Windows (console version):**
```bash
go build -o sqldashboard.exe
```

**macOS:**
```bash
go build -o sqldashboard
```

**Linux:**
```bash
go build -o sqldashboard
```

**Cross-compile for Windows from Linux/macOS:**
```bash
GOOS=windows GOARCH=amd64 go build -o sqldashboard.exe
```

## Configuration

Create a `config.json` file in the same directory as the executable. Example configuration:

```json
{
  "refreshTimeout": "1m",
  "defaultConnection": {
    "host": "localhost",
    "port": "5432",
    "user": "username",
    "password": "password",
    "database": "your-database"
  },
  "areas": [
    {
      "title": "Database Overview",
      "refreshTimeout": "30s",
      "tabs": [
        {
          "title": "Table Statistics",
          "query": "SELECT 'customers' FROM customers ORDER BY created_at LIMIT 10",
          "refreshTimeout": "1m"
        }
      ]
    }
  ]
}
```

### Configuration options

- `refreshTimeout`: Global refresh interval (e.g. "30s", "1m", "5m")
- `defaultConnection`: PostgreSQL connection parameters
- `areas`: Dashboard sections
  - Each section contains tabs with SQL queries
  - Each tab can override the refresh interval

## Usage

1. **Configure** connections and queries in `config.json`
2. **Run** the application:
   - Windows: Double-click `sqldashboard.exe` or run from command line
   - macOS/Linux: `./sqldashboard`
3. **Navigation**:
   - Left panel: Sections and tabs
   - Right panel: Query results
4. **Interaction**:
   - Adjust column widths with mouse
   - Copy cell contents (Ctrl+C)
   - Data auto-refreshes according to settings

## Troubleshooting

### Common issues

**PostgreSQL connection errors:**
- Ensure PostgreSQL server is running
- Verify connection parameters in `config.json`
- Install PostgreSQL client libraries if needed

**Missing dependencies:**
```bash
go mod tidy
go mod vendor
```

**Windows-specific:**
- For `libpq.dll` errors, install [PostgreSQL for Windows](https://www.postgresql.org/download/windows/)
- Or copy `libpq.dll` to the application directory

## License

MIT License. See [LICENSE](LICENSE) for details.
