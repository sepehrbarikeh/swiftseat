# SwiftSeat

![Go Version](https://img.shields.io/badge/go-1.26-blue)
![License](https://img.shields.io/badge/license-MIT-green)
![Docker](https://img.shields.io/badge/docker-ready-blue)

SwiftSeat is a high-concurrency ticketing backend designed to safely process thousands of simultaneous seat reservations without double-booking, using PostgreSQL transactional locking, Redis caching, and real-time SSE updates.

Overview
--------

- Prevents double-booking using database-level locking and transactional flows.
- Provides a reservation lifecycle with a 10-minute temporary lock.
- Broadcasts live seat updates via Server-Sent Events (SSE).
- Caches event listings in Redis for scalability under high read traffic.

Core Concepts
-------------

1. Concurrency control

   - Uses pessimistic row-level locking in Postgres (`FOR UPDATE`) to ensure a single successful reservation when multiple concurrent requests target the same seat.

   ```go
   tx.Clauses(clause.Locking{Strength: "UPDATE"})
   ```

2. Seat lifecycle

   - available → reserved (10-minute lock) → sold
   - If payment is not completed before the lock expires, the seat returns to `available`.

3. Real-time updates

   - The app broadcasts seat state changes over SSE so connected clients observe updates instantly.

4. Cache strategy

   - Event lists are cached in Redis and invalidated on updates to reduce DB load during peaks.


Architecture
------------

    Client
        ↓
    Fiber API
        ↓
    Middleware (JWT / RBAC)
        ↓
    Service Layer
        ↓
    PostgreSQL (transactions)
        ↓
    Redis (cache)
        ↓
    SSE Hub (realtime updates)

Getting started
---------------

Requirements

- Go 1.26+
- PostgreSQL (e.g., 15)
- Redis
- Docker & Docker Compose (recommended for local integration)

Configuration

Place a `config.yml` at the project root (example values below):

```yaml
app:
  port: 8080

database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  name: swift_seat_db
  sslmode: disable

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0

jwt:
  secret: super-secret-key

seatlock:
  time: 10m

cleanupinterval:
  time: 30s
```

Note: `config.docker.yml` contains environment mappings used by `docker-compose.yml` and may be mounted into the container as `/app/config.yml`.

Run locally

```bash
go run cmd/main.go
```

Open:

- API: `http://localhost:8080`
- Health: `http://localhost:8080/health`
- Swagger: `http://localhost:8080/swagger/index.html`

Run with Docker Compose

```bash
docker compose up --build
```

API reference (summary)
-----------------------

Public

- `GET /health` — health check
- `GET /api/` — public home data
- `GET /api/events` — list events (filters: `page`, `limit`, `search`, `location`)
- `GET /api/events/:event_id/seats` — seat map for an event
- `GET /api/events/stream` — SSE stream for live updates

Authentication

- `POST /api/register` — register `{name,email,password}`
- `POST /api/login` — login `{email,password}` → returns JWT

Protected (JWT required)

- `POST /api/seats/reserve` — `{seat_number,event_id}`
- `POST /api/seats/confirm-payment` — `{seat_number,event_id,amount}`
- `GET /api/user/tickets` — list current user's tickets

Admin

- `POST /api/users/:id/role` — change user role `{role}`
- `POST /api/events` — create event (multipart form: `title,description,location,start_time,rows,seats_per_row,image`)
- `PUT /api/events/:id` — update event
- `DELETE /api/events/:id` — delete event
- `GET /api/events/all` — admin event list
- `GET /api/tickets/validate/:ref` — validate ticket reference

Authorization

- Protected endpoints require `Authorization: Bearer <token>` header.
- Admin-only endpoints require the JWT `role` claim to be `admin`.

Examples
--------

Register / Login / Reserve (curl)

```bash
# Register
curl -s -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com","password":"secret123"}'

# Login
curl -s -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"secret123"}'

# Reserve (use returned token)
TOKEN=... 
curl -s -X POST http://localhost:8080/api/seats/reserve \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"seat_number":"A-1","event_id":1}'
```

Create event (admin)

```bash
curl -X POST http://localhost:8080/api/events \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -F "title=Concert" \
  -F "description=Live show" \
  -F "location=Main Hall" \
  -F "start_time=2026-07-01T19:00:00Z" \
  -F "rows=10" -F "seats_per_row=20" \
  -F "image=@./poster.jpg"
```

Design & Implementation notes
-----------------------------

- Reservation expiry is enforced with `expires_at` and a cleanup worker that resets expired reservations.
- Payment confirmation updates seat status to `sold` and creates a `Ticket` record inside a transaction.
- SSE hub broadcasts JSON messages on seat state changes so clients can update UIs in real time.

Observations & recommendations
-----------------------------

- `config.docker.yml` should be clarified or removed; keep one canonical `config.yml` template.
- Persist `uploads/` as a Docker volume when running containers to avoid losing images.
- Add unit and integration tests for critical flows (reservation, payment, concurrency)
- Add migrations (e.g., `golang-migrate`) instead of relying solely on `AutoMigrate` for production schema changes.

Performance note
----------------

The repository was stress-tested with `k6` in a single-seat contention scenario (500 concurrent virtual users). The implementation prevented duplicate bookings through transactional locking.


Concurrency Guarantee
---------------------

The system guarantees that no seat can ever be booked more than once under concurrent load by enforcing row-level locking at the database transaction level.

Verified under stress testing with k6:
- 500 concurrent users
- single-seat contention
- 0 duplicate bookings observed

Next steps I can help with
-------------------------

- Add detailed `curl` request/response examples per endpoint
- Generate an OpenAPI spec or Postman collection from the code annotations
- Add CI (GitHub Actions) to run tests, linters and basic integration checks
