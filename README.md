# 🏆 BidZy - Real-Time Auction Platform

<div align="center">

![BidZy Logo](https://img.shields.io/badge/BidZy-Auction%20Platform-blue?style=for-the-badge&logo=data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTEyIDJMMTMuMDkgOC4yNkwyMCA5TDEzLjA5IDE1Ljc0TDEyIDIyTDEwLjkxIDE1Ljc0TDQgOUwxMC45MSA4LjI2TDEyIDJaIiBzdHJva2U9IndoaXRlIiBzdHJva2Utd2lkdGg9IjIiLz4KPC9zdmc+)

**A modern, scalable, and feature-rich real-time auction platform built with Go**

[![Go Version](https://img.shields.io/badge/Go-1.24.2-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15.0-336791?style=flat-square&logo=postgresql)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7.2-DC382D?style=flat-square&logo=redis)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Supported-2496ED?style=flat-square&logo=docker)](https://www.docker.com/)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)

[🚀 Quick Start](#-quick-start) • [📚 Documentation](#-api-documentation) • [🏗️ Architecture](#️-architecture) • [🤝 Contributing](#-contributing)

</div>

---

## 📖 Table of Contents

- [🌟 Features](#-features)
- [🏗️ Architecture](#️-architecture)
- [🚀 Quick Start](#-quick-start)
- [⚙️ Configuration](#️-configuration)
- [🐳 Docker Deployment](#-docker-deployment)
- [📚 API Documentation](#-api-documentation)
- [🧪 Testing](#-testing)
- [🚀 Production Deployment](#-production-deployment)
- [🔧 Development](#-development)
- [🤝 Contributing](#-contributing)
- [📄 License](#-license)

## 🌟 Features

### 🔥 Core Features
- **Real-time Bidding**: WebSocket-powered live bidding with instant updates
- **User Authentication**: Secure JWT-based authentication with Google OAuth integration
- **Advanced Search**: Category-based filtering and advanced auction discovery
- **File Upload**: S3-compatible image upload for auction items
- **Email Notifications**: Automated email alerts for auction events
- **Robust Database**: PostgreSQL with optimized indexing and triggers
- **Rate Limiting**: Built-in protection against API abuse
- **Graceful Shutdown**: Proper cleanup of resources and connections

### 🛡️ Security & Performance
- **JWT Authentication**: Secure token-based authentication
- **Password Hashing**: bcrypt password encryption
- **CORS Protection**: Configurable cross-origin resource sharing
- **Rate Limiting**: Redis-based rate limiting middleware
- **Database Migrations**: Version-controlled database schema management
- **Connection Pooling**: Optimized database connection management

### 🔄 Real-time Features
- **Live Bid Updates**: Instant bid notifications via WebSockets
- **Auction Status Tracking**: Real-time auction state management
- **User Activity Monitoring**: Live user engagement tracking
- **Automated Scheduling**: Cron-based auction lifecycle management

## 🏗️ Architecture

BidZy follows a clean, modular architecture with clear separation of concerns:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │   Load Balancer │    │   Monitoring    │
│   (Web/Mobile)  │◄──►│   (Nginx/ALB)   │◄──►│   (Metrics)     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 ▼
                    ┌─────────────────────┐
                    │    BidZy Server     │
                    │   (Go Application)  │
                    └─────────────────────┘
                                 │
                ┌────────────────┼────────────────┐
                ▼                ▼                ▼
       ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
       │ PostgreSQL  │  │    Redis    │  │   AWS S3    │
       │ (Database)  │  │   (Cache)   │  │  (Storage)  │
       └─────────────┘  └─────────────┘  └─────────────┘
```

### 📁 Project Structure

```
BidZy/
├── cmd/                    # Application entry points
│   ├── server/            # Main server application
│   └── seed/              # Database seeding utility
├── internal/              # Private application code
│   ├── handler/           # HTTP handlers and routing
│   ├── middleware/        # HTTP middleware (auth, CORS, rate limiting)
│   ├── migrations/        # Database migrations
│   ├── scheduler/         # Background job scheduling
│   ├── service/           # Business logic layer
│   │   ├── auction/       # Auction management & WebSocket
│   │   ├── auth/          # Authentication & authorization
│   │   ├── bid/           # Bidding logic
│   │   ├── category/      # Category management
│   │   ├── mail/          # Email services
│   │   └── user/          # User management
│   └── store/             # Data access layer
├── pkg/                   # Public packages
│   ├── types/             # Shared data types
│   └── utils/             # Utility functions
└── database/              # Database files (development)
```

### 🎯 Service Layer Architecture

- **Authentication Service**: JWT token management, OAuth integration
- **Auction Service**: Core auction logic, WebSocket management
- **Bidding Service**: Bid validation, real-time updates
- **User Service**: Profile management, statistics
- **Category Service**: Category-based organization
- **Mail Service**: Notification system, email templates
- **File Service**: S3 integration for media uploads

## 🚀 Quick Start

### Prerequisites

- **Go 1.24.2+**
- **PostgreSQL 15+**
- **Redis 7.2+**
- **Docker & Docker Compose** (recommended)

### 🐳 Docker Setup (Recommended)

1. **Clone the repository**
```bash
git clone https://github.com/LikhithMar14/BidZy.git
cd BidZy
```

2. **Start services with Docker Compose**
```bash
docker-compose up -d
```

3. **Set up environment variables**
```bash
cp .env.example .env
# Edit .env with your configuration
```

4. **Run the application**
```bash
go run cmd/server/main.go
```

### 🛠️ Manual Setup

1. **Install dependencies**
```bash
go mod tidy
```

2. **Set up PostgreSQL**
```bash
# Create database
createdb auction-db

# Set environment variables
export DB_ADDR="postgres://user:password@localhost:5460/auction-db?sslmode=disable"
```

3. **Set up Redis**
```bash
# Start Redis server
redis-server

# Or use Docker
docker run -d -p 6379:6379 redis:7.2-alpine
```

4. **Configure environment**
```bash
export JWT_SECRET="your-super-secret-jwt-key"
export GOOGLE_CLIENT_ID="your-google-oauth-client-id"
export GOOGLE_CLIENT_SECRET="your-google-oauth-secret"
export S3_BUCKET_NAME="your-s3-bucket"
export SMTP_HOST="smtp.gmail.com"
export SMTP_PORT="587"
export SMTP_USER="your-email@gmail.com"
export SMTP_PASS="your-app-password"
```

5. **Run migrations and start server**
```bash
go run cmd/server/main.go
```

The server will start on `http://localhost:8080` 🎉

## ⚙️ Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | Server port | `8080` | No |
| `DB_ADDR` | PostgreSQL connection string | - | Yes |
| `JWT_SECRET` | JWT signing secret | - | Yes |
| `GOOGLE_CLIENT_ID` | Google OAuth client ID | - | Yes |
| `GOOGLE_CLIENT_SECRET` | Google OAuth secret | - | Yes |
| `GOOGLE_REDIRECT_URL` | OAuth redirect URL | - | Yes |
| `S3_BUCKET_NAME` | AWS S3 bucket name | - | Yes |
| `AWS_REGION` | AWS region | `us-east-1` | No |
| `SMTP_HOST` | SMTP server host | - | Yes |
| `SMTP_PORT` | SMTP server port | `587` | No |
| `SMTP_USER` | SMTP username | - | Yes |
| `SMTP_PASS` | SMTP password | - | Yes |

### Database Configuration

The application uses PostgreSQL with the following optimizations:
- **Connection pooling** for better performance
- **Automatic migrations** on startup
- **Indexed columns** for fast queries
- **Foreign key constraints** for data integrity
- **Triggers** for automatic timestamp updates

## 🐳 Docker Deployment

### Development Environment

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

### Production Environment

```bash
# Use production compose file
docker-compose -f docker-compose-production.yml up -d
```

## 📚 API Documentation

### 🔐 Authentication Endpoints

#### Register User
```http
POST /api/v1/users/register
Content-Type: application/json

{
  "user_name": "johndoe",
  "email": "john@example.com",
  "password": "securepassword"
}
```

#### Login
```http
POST /api/v1/users/login
Content-Type: application/json

{
  "email": "john@example.com",
  "password": "securepassword"
}
```

#### Google OAuth
```http
GET /api/v1/auth/google/login
GET /api/v1/auth/google/callback
```

### 🏆 Auction Endpoints

#### Create Auction
```http
POST /api/v1/auctions
Authorization: Bearer <jwt_token>
Content-Type: multipart/form-data

{
  "title": "Vintage Guitar",
  "description": "Beautiful 1960s acoustic guitar",
  "starting_price": 500.00,
  "start_date": "2024-12-01T10:00:00Z",
  "end_date": "2024-12-07T22:00:00Z",
  "categories": [1, 2],
  "image": <file>
}
```

#### Get Auctions
```http
GET /api/v1/auctions?category=1&status=ACTIVE&page=1&limit=20
```

#### Get Auction Details
```http
GET /api/v1/auctions/{id}
```

### 💰 Bidding Endpoints

#### Place Bid
```http
POST /api/v1/auctions/{id}/bids
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "amount": 550.00
}
```

#### Get Auction Bids
```http
GET /api/v1/auctions/{id}/bids
```

### 📊 User Endpoints

#### Get User Profile
```http
GET /api/v1/users/{id}
Authorization: Bearer <jwt_token>
```

#### Get User Statistics
```http
GET /api/v1/users/{id}/stats
Authorization: Bearer <jwt_token>
```

### 🗂️ Category Endpoints

#### Get Categories
```http
GET /api/v1/categories
```

#### Create Category
```http
POST /api/v1/categories
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "name": "Electronics"
}
```

### 🔌 WebSocket Connection

Connect to real-time auction updates:

```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/auctions/{auction_id}/ws');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('New bid:', data);
};
```

## 🧪 Testing

### Run Tests
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/service/auction/...
```

### Database Testing
```bash
# Run integration tests
go test -tags=integration ./...
```

### Load Testing
```bash
# Install hey (HTTP load testing tool)
go install github.com/rakyll/hey@latest

# Test auction endpoint
hey -n 1000 -c 50 http://localhost:8080/api/v1/auctions
```

## 🚀 Production Deployment

### 🌐 AWS Deployment

1. **Set up RDS PostgreSQL**
2. **Configure ElastiCache Redis**
3. **Set up S3 bucket for file storage**
4. **Deploy using ECS or EKS**

### 🐳 Docker Production Build

```dockerfile
# Multi-stage build for production
FROM golang:1.24.2-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main cmd/server/main.go

FROM alpine:3.18
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
CMD ["./main"]
```

### 📊 Monitoring & Logging

- **Structured Logging**: Using Zap logger for performance
- **Health Checks**: Built-in health check endpoints
- **Metrics**: Prometheus-compatible metrics
- **Tracing**: OpenTelemetry support

## 🔧 Development

### Code Style
- Follow Go best practices and conventions
- Use `gofmt` for code formatting
- Use `golint` for linting
- Write meaningful tests with good coverage

### Database Migrations
```bash
# Create new migration
goose -dir ./internal/migrations create migration_name sql

# Apply migrations
goose -dir ./internal/migrations postgres "connection_string" up

# Rollback migration
goose -dir ./internal/migrations postgres "connection_string" down
```

### Adding New Features

1. **Service Layer**: Implement business logic in `internal/service/`
2. **Store Layer**: Add data access methods in `internal/store/`
3. **Handler Layer**: Create HTTP handlers in `internal/handler/`
4. **Types**: Define data structures in `pkg/types/`
5. **Tests**: Write comprehensive tests

## 🤝 Contributing

We love contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Workflow

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/amazing-feature`
3. **Make changes and add tests**
4. **Ensure tests pass**: `go test ./...`
5. **Commit changes**: `git commit -am 'Add amazing feature'`
6. **Push to branch**: `git push origin feature/amazing-feature`
7. **Open a Pull Request**

### 🐛 Bug Reports

Found a bug? Please open an issue with:
- Clear description of the problem
- Steps to reproduce
- Expected vs actual behavior
- System information (Go version, OS, etc.)

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

<div align="center">

**[⬆ Back to Top](#-bidzy---real-time-auction-platform)**

Made with ❤️ by the BidZy Team

[![GitHub Stars](https://img.shields.io/github/stars/LikhithMar14/BidZy?style=social)](https://github.com/LikhithMar14/BidZy)
[![GitHub Forks](https://img.shields.io/github/forks/LikhithMar14/BidZy?style=social)](https://github.com/LikhithMar14/BidZy/fork)

</div> 