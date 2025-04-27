# Crypto Wallet Management API

A Go service for managing cryptocurrency wallet transactions with Redis caching and PostgreSQL storage.

## Features ✨

- 💰 Deposit/Withdraw/Transfer operations
- 📈 Real-time balance tracking
- 🔒 ACID-compliant transactions
- ⚡ Redis caching layer
- 📊 Transaction history pagination

## Tech Stack 🛠️

| Layer        | Technology       |
|--------------|------------------|
| Language     | Go 1.20+         |
| Web Framework| Gin              |
| Database     | PostgreSQL       |
| Cache        | Redis            |
| ORM          | pgx/pgxpool      |
| Testing      | Testify + GoMock |

## Getting Started 🚀

### Installation
#### Prerequisites
- Go 1.20+
- PostgreSQL 15
- Redis 7

#### Setup
1. Clone the repository
```bash
git clone git@github.com:juinhong/wallet_app.git
cd wallet_app
```

2. Install dependencies
```bash
go mod download
```

3. Set up your PostgreSQL and Redis databases

**PostgreSQL**:
```bash
# Create database and tables
psql -U postgres -c "CREATE DATABASE wallet_db;"
psql -U postgres -d wallet_db -c "
CREATE TABLE wallets (
    user_id VARCHAR(255) PRIMARY KEY,
    balance DECIMAL NOT NULL DEFAULT 0.0
);

CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    from_user_id VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL,
    amount DECIMAL NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    to_user_id VARCHAR(255)
);

-- Create optimized indexes
CREATE INDEX idx_transactions_user_ts ON transactions USING btree (user_id, timestamp DESC);
CREATE INDEX idx_transactions_receiver ON transactions USING btree (receiver_id);
CREATE INDEX idx_wallets_balance ON wallets USING btree (balance);
CREATE INDEX idx_transactions_user_type ON transactions USING btree (user_id, type);
```

**Redis**:
```bash
# Install via Homebrew
brew install redis

# Configure automatic startup
brew services start redis

# Edit config for persistence
nano /usr/local/etc/redis.conf
"""
save 900 1
save 300 10
save 60 10000
"""

# Start with password protection
redis-server /usr/local/etc/redis.conf --requirepass "your_strong_password"

# Verify connection
redis-cli -a "your_strong_password" ping
```

4. Update the database connection details in `internal/config/config.go`
5. Run the server
```bash
go run cmd/server/main.go
```

## API Documentation
### Deposit Funds
**Endpoint**  
`POST /api/v1/wallets/{userID}/deposit`

**Request Body**
```json
{
  "amount": 100.50
}
```

**Request Body**

Status: 200 OK (empty body)

Error: 400 Bad Request or 500 Internal Server Error
```json
{
  "error": "Invalid amount"
}
```

### Withdraw Funds
**Endpoint**
`POST /api/v1/wallets/{userID}/withdraw`

**Request Body**
```json
{
  "amount": 50.25
}
```

**Response**

Status: 200 OK (empty body)

Error: 400 Bad Request or 500 Internal Server Error
```json
{
  "error": "Insufficient funds"
}
```

### Transfer Funds
**Endpoint**
`POST /api/v1/wallets/{userID}/transfer`

**Request Body**
```json
{
  "amount": 25.00,
  "receiver_id": "recipient123"
}
```

**Response**

Status: 200 OK (empty body)

Error: 400 Bad Request or 500 Internal Server Error
```json
{
  "error": "Insufficient funds"
}
```

### Get Balance
**Endpoint**
`GET /api/v1/wallets/{userID}/balance`

**Response**
Status: 200 OK
```json
{
  "balance": 75.00
}
```

### Get Transaction History
**Endpoint**
`GET /api/v1/wallets/{userID}/transactions`

**Request Body**
```json
{
  "page": 1,
  "limit": 10
}
```

**Response**

Status: 200 OK

page: The current page number

limit: The number of transactions per page

transactions: An array of transaction objects

total: The total number of transactions
```json
{
  "page": 1,
  "limit": 10,
  "transactions": [
    {
      "id": 1,
      "type": "deposit",
      "amount": 100.50,
      "timestamp": "2023-10-10T12:00:00Z"
    }
  ],
  "total": 1
}
```

### Error Handling

❗ Any database scan failure will return 500 Internal Server Error

🔒 Strict data validation during result parsing

🛑 Partial responses are never returned

Status: 400 Bad Request or 500 Internal Server Error
```json
{
  "error": "Invalid amount"
}
```

## Project Structure 📁
```
.
├── cmd/
│   └── server/
│       └── main.go # Application entry point (server configuration)
├── internal/
│   ├── config/
│       └── config.go # Configuration loading (DB, Redis, etc.)
│   ├── handlers/
│   │   └── wallet.go # HTTP handlers (Gin routes and controllers)
│   │   └── logging.go # Middleware for request logging
│   ├── models/
│   │   └── transaction.go # Data structures (DB schema mappings)
│   ├── repositories/
│   │   └── postgres/
│   │   │   └── wallet_repository.go # Database operations (CRUD)
│   │   └── redis/
│   │       └── cache_repository.go # Redis cache operations
│   └── services/
│       └── wallet_service.go # Business logic (transaction orchestration)
├── go.mod # Go module dependencies
├── go.sum
└── README.md
```
**Key Review Areas**:
1. **Business Logic**: Start with `services/wallet_service.go` for core transaction logic
2. **Database Interactions**: `repositories/postgres/wallet_repository.go`
3. **API Endpoints**: `handlers/wallet.go` (Gin route handlers)
4. **Cache Strategy**: `repositories/redis/cache_repository.go`
5. **Error Handling**: See `logging.go` and error returns in service layer
6. **Transactions**: Look for `SELECT FOR UPDATE` in repository methods

### Key Design Decisions 💡
Caching Strategy:
- Lazy loading pattern for balance checks
- TTL-based cache invalidation
- Write-through cache for balance updates
  - Why This Fits My Design:
    
    💰 Balance Consistency: Ensures cache always matches database state
    
    ⚡ Fast Reads: Subsequent balance checks use Redis cache
    
    🔒 Data Safety: No stale financial data in cache after updates
    
  - Tradeoffs:
    
    ➕ Strong consistency between cache and DB
    
    ➕ Simplified cache invalidation
    
    ➖ Slightly slower writes (waits for both DB and cache updates)

Transaction Management:
- Database-level locking (SELECT FOR UPDATE)
- Database Indexing:
  
  - | Table        | Index Name                       | Columns                 | Purpose                              |
    |--------------|----------------------------------|-------------------------|--------------------------------------|
    | transactions | idx_transactions_user_ts         | (user_id, timestamp)    | Fast transaction history pagination  |
    | transactions | idx_transactions_receiver        | receiver_id             | Quick transfer recipient lookups     |
    | wallets      | idx_wallets_balance              | balance                 | Accelerate balance check operations  |
    | transactions | idx_transactions_user_type       | (user_id, type)         | Efficient transaction type filtering |

```sql
-- PostgreSQL Index Definitions
CREATE INDEX idx_transactions_user_ts ON transactions USING btree (user_id, timestamp DESC);
CREATE INDEX idx_transactions_receiver ON transactions USING btree (receiver_id);
CREATE INDEX idx_wallets_balance ON wallets USING btree (balance);
CREATE INDEX idx_transactions_user_type ON transactions USING btree (user_id, type);
```

Error Handling:
- Structured error responses
- Error wrapping with stack traces
- Graceful shutdown handling
- Meaningful log messages

### Development Timeline ⏳
Task: Time Spent

Technical Design: 2 hours

Code Implementation: 10 hours
- Configuration: 1 hour
- API Endpoints: 1 hour
- Service Layer: 1 hour
- Repository Layer: 2 hours
  - PostgreSQL: 1 hour
  - Redis: 1 hour
- Error Handling: 1 hour
- Logging: 1 hour
- Unit Testing: 3 hours

Testing: 3 hours
- Database Setup: 1 hour
- Redis Setup: 1 hour
- Testing Endpoints: 1 hour

Documentation: 2 hours

## Future Improvements 🔮
Security:
- JWT Authentication

Rate Limiting:
- Implement rate limiting for API endpoints

Monitoring:
- Prometheus Metrics for APIs:
  - APIs QPS
  - APIs latency
  - APIs error rates
  - CPU/RAM usage
- Prometheus Metrics for Database:
  - Database connections
  - Database queries
  - Failed DB queries
- Prometheus Metrics for Cache:
  - Cache connections
  - Cache size
  - Cache hits/misses
  - Failed cache operations
  - Failed cache evictions

Health Checks
- Persist logging to remote storage
- Persist metrics to remote storage
- Persist traces to remote storage

Features:
- Multi-Currency Support
- Webhook Notifications
- Admin Dashboard

Stress Testing:
- Set up a stress testing tool (e.g., Apache Bench)
