# GEO Tracker

GEO Tracker is a Go-based CLI tool and API server designed to monitor brand visibility across multiple AI providers (Claude, ChatGPT, Perplexity, and Gemini).

## Prerequisites

- Go 1.23+
- MySQL 8.0+
- AI Provider API Keys

## Installation

```bash
go build -o geo-tracker .
```

## Setup

1. **Configure**: Copy `config.yaml` to `config.local.yaml` and add your API keys and database credentials.
2. **Validate & Migrate**: Run the validation command to check your configuration and initialize the database schema.
   ```bash
   ./geo-tracker config validate
   ```
3. **Seed Prompts**: Import the initial set of curated prompts.
   ```bash
   ./geo-tracker prompts import prompts/seed.yaml
   ```

## CLI Commands

### Probing
*   **Run All**: Execute all active prompts across enabled providers for configured brands.
    ```bash
    ./geo-tracker run
    ```
    *   `--brands stringSlice`: Override brands from config.
    *   `--providers stringSlice`: Run only specific providers.
    *   `--dry-run`: Probe providers without saving to DB.
    *   `--verbose`: Print raw AI responses to stdout.

### Results
*   **Summary**: Show mention rates and sentiment for the latest run.
    ```bash
    ./geo-tracker results summary
    ```
    *   `--run-id uint`: View a specific run.
    *   `--brand string`: Filter by brand.
    *   `--provider string`: Filter by provider.

### Prompt Management
*   **List**: Show all active prompts.
    ```bash
    ./geo-tracker prompts list
    ```
*   **Import**: Bulk import prompts from a YAML file.
    ```bash
    ./geo-tracker prompts import <path_to_yaml>
    ```

### Server
*   **Serve**: Start the JSON API server.
    ```bash
    ./geo-tracker serve
    ```

## API Endpoints

The API serves JSON at `http://localhost:8080/api` (by default).

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/api/health` | Service and Database status |
| `GET` | `/api/prompts` | List all active prompts |
| `GET` | `/api/runs` | List execution runs (latest first) |
| `GET` | `/api/runs/{id}/results` | Detailed results for a specific run |

## Logging

Logs are emitted to `stderr` in structured JSON format. 
To run commands without seeing logs in the terminal:
```bash
./geo-tracker <command> 2>/dev/null
```
To change log level, set `app.log_level` in config or use environment variables:
`GEOTRACKER_APP_LOG_LEVEL=debug ./geo-tracker run`
