# go-event-analytics

## Overview
`go-event-analytics` is a Go-based application designed to analyze event data. It provides insights into event patterns and allows users to manage and visualize event-related analytics.

## Features
- Analyze event data with custom metrics.
- Upload event files for processing.
- View analytics in a structured format.

## Requirements
- Go 1.19 or newer
- A PostgreSQL database (or your preferred database)
- [Optional] Docker for containerized setup

## Installation

### 1. Clone the Repository
```bash
git clone https://github.com/yourusername/go-event-analytics.git
cd go-event-analytics
```

### 2. Install Dependencies
```bash
go mod tidy
```

### 3. Set Up Environment Variables by renaming the .example.env to .env
```bash
DB_HOST=your_database_host
DB_PORT=your_database_port
DB_USER=your_database_user
DB_PASSWORD=your_database_password
DB_NAME=your_database_name
```

### 4. Run the Application
```bash
go run cmd/main.go
```

### 5. Build the Application
```bash
go build -o go-event-analytics .
```

### 6. Run Tests
```bash
go test ./...
```