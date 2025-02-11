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

### spin setup [app-name]

Initialize a new application with Spin configuration.

```bash
spin setup myapp              # Setup new app in myapp directory
spin setup myapp --repo=org/name  # Setup with specific repository
spin setup . --force         # Setup in current directory
```

Flags:

- `--repo`: Specify repository in format organization/name
- `--force`: Force overwrite existing configuration

The setup command will:

- Create project directory (if needed)
- Initialize spin.config.json
- Detect project type and configure accordingly (e.g., Rails applications)
- Set up repository information

### spin config

Manage Spin configuration settings.

```bash
spin config show              # Show current configuration
spin config set-org myorg     # Set default organization
```

Subcommands:

- `show`: Display current configuration
- `set-org [organization]`: Set default GitHub organization for project setup

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
    "setup": "bundle install && rails db:setup",
    "start": "rails server",
    "test": "rails test"
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
    "services": {}
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

1. Create a `spin.config.json` in your project
2. Add a `Procfile.dev` if you need multiple processes
3. Run `spin up` to start your development environment
4. Use `spin ps` to monitor running processes
5. View logs with `spin logs [process]`
6. Debug with `spin debug [process]` when needed
7. Stop everything with `spin down`
