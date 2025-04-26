# Crypto Wallet Management API

A production-ready Go service for managing cryptocurrency wallet transactions with Redis caching and PostgreSQL storage.

## Features ✨

- 💰 Deposit/Withdraw/Transfer operations
- 📈 Real-time balance tracking
- 🔒 ACID-compliant transactions
- ⚡ Redis caching layer
- 📊 Transaction history pagination

## Tech Stack 🛠️

| Layer        | Technology          |
|--------------|---------------------|
| Language     | Go 1.20+           |
| Web Framework| Gin                 |
| Database     | PostgreSQL 15       |
| Cache        | Redis 7             |
| ORM          | pgx/pgxpool         |
| Testing      | Testify + GoMock    |

## Getting Started 🚀

### Prerequisites
- Go 1.20+
- PostgreSQL 15
- Redis 7

### Installation

API Endpoints 📡
Method
Endpoint
Description
POST
/api/v1/wallets/{userID}/deposit
Add funds
POST
/api/v1/wallets/{userID}/withdraw
Withdraw funds
POST
/api/v1/wallets/{userID}/transfer
Transfer funds
GET
/api/v1/wallets/{userID}/balance
Get balance

### Key Design Decisions 💡
Caching Strategy:
Lazy loading pattern for balance checks
TTL-based cache invalidation
Write-through cache for balance updates
Transaction Management:
Database-level locking (SELECT FOR UPDATE)
Automatic transaction retries
Context-aware timeouts
Error Handling:
Structured error responses
Error wrapping with stack traces
Graceful shutdown handling

### Development Timeline ⏳
Task
Time Spent
Core Architecture
3 hours
Transaction Logic
4 hours
Testing
2 hours
Documentation
1 hour
Future Improvements 🔮
Security:
JWT Authentication
Rate Limiting
Request Validation
Monitoring:
Prometheus Metrics
Health Checks
Structured Logging
Features:
Multi-Currency Support
Webhook Notifications
Admin Dashboard
