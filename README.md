# pollkit

Agentic-first poll and survey service. Create polls, collect votes, and view results. Plain text API, agent-driven, single Go binary with JSON file storage.

## Quick Start

```bash
# Build
make build

# Run (defaults to :7777, data in pollkit.json)
./pollkit

# Get help
curl http://localhost:7777/help
```

## Auth Flow

```bash
# 1. Request OTP (code logged to stderr if no SMTP configured)
curl -X POST http://localhost:7777/auth/request -d 'email=agent@example.com'
# stderr: [OTP] agent@example.com: code=123456 workspace=ws_abc12

# 2. Verify OTP, get bearer token
curl -X POST http://localhost:7777/auth/verify -d 'email=agent@example.com&code=123456'
# token=abc123... workspace=ws_abc12 email=agent@example.com
```

## Create a Poll

```bash
curl -X POST http://localhost:7777/polls \
  -H 'Authorization: Bearer <token>' \
  -d 'question=Best language for agents?&options=Go,Python,Rust&public=true'
# handle=poll_x1y2z question=Best language for agents? options=3 multiple=false public=true opt1=Go opt2=Python opt3=Rust
```

## Vote

```bash
curl -X POST http://localhost:7777/vote \
  -d 'poll=poll_x1y2z&option=opt1&voter=alice'
# handle=vote_a1b2c poll=poll_x1y2z option=opt1 voter=alice
```

## View Results

```bash
curl http://localhost:7777/results/poll_x1y2z
# poll=poll_x1y2z total=1
# option_id=opt1 label=Go count=1 percent=100.0
# option_id=opt2 label=Python count=0 percent=0.0
# option_id=opt3 label=Rust count=0 percent=0.0
```

## JSON Output

Add `?format=json` or `Accept: application/json`:

```bash
curl -H 'Accept: application/json' http://localhost:7777/results/poll_x1y2z
```

## Configuration

| Flag    | Env              | Default        | Description                    |
|---------|------------------|----------------|--------------------------------|
| -addr   | POLLKIT_ADDR     | :7777          | Listen address                 |
| -db     | POLLKIT_DB       | pollkit.json   | Data file path                 |
| -secret | POLLKIT_SECRET   | (auto-generated)| Token signing secret          |
| -smtp   | POLLKIT_SMTP     | (empty)        | SMTP URL for OTP emails        |

## API Endpoints

| Method | Path              | Auth  | Description                    |
|--------|-------------------|-------|--------------------------------|
| GET    | /help             | No    | Operating manual for agents    |
| POST   | /auth/request     | No    | Request OTP via email          |
| POST   | /auth/verify      | No    | Verify OTP, get bearer token   |
| POST   | /polls            | Yes   | Create a poll                  |
| GET    | /polls            | Yes   | List your polls               |
| GET    | /polls/<handle>   | Yes*  | Get poll details              |
| DELETE | /polls/<handle>   | Yes   | Delete poll and all votes     |
| POST   | /vote             | Yes*  | Cast a vote                   |
| DELETE | /vote             | Yes*  | Remove a vote                 |
| GET    | /results/<handle> | Yes*  | View poll results             |

*Public polls allow access without auth.

## Build

```bash
make build    # CGO_ENABLED=0, single binary
make test     # go test -race
make vet      # go vet
```

## License

MIT
