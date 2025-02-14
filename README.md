# Spin CLI

Spin is a development environment manager that helps you run and manage multiple processes in your development workflow.

## Installation

```bash
go install github.com/afomera/spin@latest
```

## Requirements

- tmux (required for process management)

## Commands

### spin up [app-name]

Start the development environment for an application.

```bash
spin up           # Start the app in the current directory
spin up myapp     # Start the app in the myapp directory
```

The command reads configuration from `spin.config.json` and starts all processes defined in your Procfile.dev.

### spin down

Stop all running processes and clean up the development environment.

```bash
spin down         # Stop all processes
```

### spin ps

List all running processes and their status.

```bash
spin ps           # Show process list
```

### spin logs [process-name]

View the output logs for a specific process.

```bash
spin logs web     # View web process logs
```

### spin debug [process-name]

Attach to a process in debug mode (useful for interactive debugging sessions).

```bash
spin debug web    # Debug web process
```

### spin init [app-name]

Initialize a new application with Spin configuration.

```bash
spin init myapp              # Initialize new app in myapp directory
spin init myapp --repo=org/name  # Initialize with specific repository
spin init . --force         # Initialize in current directory
```

Flags:

- `--repo`: Specify repository in format organization/name
- `--force`: Force overwrite existing configuration

The init command will:

- Create project directory (if needed)
- Initialize spin.config.json
- Detect project type and configure accordingly (e.g., Rails applications)
- Set up repository information

### spin scripts

Manage and run scripts defined in your configuration.

```bash
# List available scripts
spin scripts list           # Show all available scripts

# Run a script
spin scripts run setup     # Run the setup script
spin scripts run test      # Run the test script

# Run with environment variables
spin scripts run test --env=NODE_ENV=test --env=DEBUG=true

# Run in specific directory
spin scripts run build --workdir=/path/to/dir

# Skip hook errors
spin scripts run deploy --skip-hook-error
```

Flags:

- `--env`: Set environment variables (can be used multiple times)
- `--workdir`: Set working directory for script execution
- `--skip-hook-error`: Continue even if hooks fail

Common scripts also have shorthand commands:

- `spin setup` - Run setup script
- `spin test` - Run test script
- `spin server` - Start development server

### spin config

Manage Spin configuration settings.

```bash
spin config show              # Show current configuration
spin config set-org myorg     # Set default organization
```

Subcommands:

- `show`: Display current configuration
- `set-org [organization]`: Set default GitHub organization for project setup

### spin services

Manage Docker-based services for your application.

```bash
# List all services and their status
spin services list           # Show all services with status and health
spin services start redis    # Start a specific service
spin services stop redis     # Stop a specific service
spin services restart redis  # Restart a service
spin services logs redis     # View service logs
spin services logs redis -f  # Stream logs continuously
spin services logs redis -n 100  # Show last 100 lines

# View detailed service information
spin services info redis     # Show detailed info including health and uptime

# Service configuration
spin services add           # Add a new service interactively
spin services remove redis  # Remove a service
spin services edit redis    # Edit service configuration
spin services export redis  # Export service configuration
spin services import redis-config.json  # Import service configuration

# Service maintenance
spin services cleanup volumes  # Clean up unused volumes
spin services update redis    # Update service to latest version
spin services update redis --version 7.0  # Update to specific version
spin services stats          # View resource usage (CPU, Memory)
```

Flags:

- `--follow, -f`: Follow log output
- `--tail, -n`: Number of lines to show from logs
- `--remove-volumes`: Remove associated volumes when removing service
- `--version`: Specify version when updating service
- `--name`: Service name for import (defaults to filename)

## Configuration

### spin.config.json

The main configuration file for your application. Here's an example of a Rails application configuration:

```json
{
  "name": "test_rails",
  "version": "1.0.0",
  "type": "rails",
  "repository": {
    "organization": "afomera",
    "name": "test_rails"
  },
  "dependencies": {
    "services": ["sqlite3"],
    "tools": ["ruby", "bundler"]
  },
  "scripts": {
    "setup": {
      "command": "bundle install",
      "description": "Install dependencies",
      "hooks": {
        "pre": {
          "command": "asdf install ruby",
          "description": "Install Ruby version",
          "env": {
            "RUBY_VERSION": "3.2.2"
          }
        },
        "post": {
          "command": "bundle exec rails db:setup",
          "description": "Set up database"
        }
      }
    },
    "test": {
      "command": "bundle exec rspec",
      "description": "Run tests",
      "hooks": {
        "pre": {
          "command": "bundle exec rails db:test:prepare",
          "description": "Prepare test database"
        }
      }
    },
    "server": {
      "command": "bundle exec rails server",
      "description": "Start Rails server",
      "hooks": {
        "pre": {
          "command": "bundle exec rails db:prepare",
          "description": "Prepare database"
        }
      }
    }
  },
  "env": {
    "development": {}
  },
  "rails": {
    "ruby": {
      "version": "3.4.1"
    },
    "database": {
      "type": "sqlite3",
      "settings": {
        "database": "storage/development.sqlite3"
      }
    },
    "rails": {
      "version": "8.0.1"
    },
    "services": {
      "postgres": {
        "type": "docker",
        "image": "postgres:17",
        "port": 5432,
        "environment": {
          "POSTGRES_USER": "postgres",
          "POSTGRES_PASSWORD": "postgres"
        },
        "volumes": {
          "data": "/var/lib/postgresql/data"
        },
        "healthCheck": {
          "command": ["pg_isready", "-U", "postgres"],
          "interval": "10s",
          "timeout": "5s",
          "retries": 3,
          "startPeriod": "10s"
        }
      },
      "redis": {
        "type": "docker",
        "image": "redis:7",
        "port": 6379,
        "volumes": {
          "data": "/data"
        }
      }
    }
  }
}
```

The configuration includes:

- Project metadata (name, version, type)
- Repository information
- Dependencies (required services and tools)
- Common scripts (setup, start, test)
- Environment variables
- Rails-specific settings (Ruby version, database config, Rails version)

### Procfile.dev

Define additional processes to run alongside your main application:

```
web: bundle exec rails server -p 3000
worker: bundle exec sidekiq
css: yarn build:css --watch
js: yarn build --watch
```

## Process Management

Spin uses tmux to manage processes, providing:

- Process isolation
- Output capture and logging
- Interactive debugging capabilities
- Clean process termination

Each process runs in its own tmux session, with logs stored in `~/.spin/output/`.

## Development Workflow

1. Initialize your project: `spin init myapp`
2. Add a `Procfile.dev` if you need multiple processes
3. Run `spin setup` to install dependencies
4. Run `spin up` to start your development environment
5. Use `spin ps` to monitor running processes
6. View logs with `spin logs [process]`
7. Debug with `spin debug [process]` when needed
8. Stop everything with `spin down`
