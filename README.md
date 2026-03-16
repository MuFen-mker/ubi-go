# Ubi-Go

Welcome to Ubi-Go! This is a Go backend that talks to Ubisoft’s services so you don’t have to. It grabs player stats, profiles, and lets you report players (if you’re feeling spicy). It’s got a Python client, runs in Docker, and tries to make your life easier if you’re building tools around Ubisoft games. Disclaimer, I did not write any unit tests, I got lazy and I needed something done fast. I might add them in the future.

---

## What’s Cool Here?

- **Get Ubisoft player stats and profiles** (for games like The Division 2)
- **Handles login/session stuff** so you don’t have to keep re-logging
- **Batch stats**: Ask for a bunch of players at once (it’ll split big requests for you)
- **Caches stuff** with Redis (if you want)
- **Logs requests** and gives you clear error messages
- **Works with Docker** (just run it, no Go knowledge needed)
- **Python client** included for easy scripting

---

## Quick Start (for Lazy People)

1. Clone this repo and copy the sample env:
	 ```bash
	 git clone <repo-url>
	 cd ubi-go
	 cp env_sample .env  # Edit .env with your Ubisoft login info
	 ```
2. Build and run with Docker Compose:
	 ```bash
	 docker-compose up --build
	 ```
3. The API will be at [http://localhost:8080](http://localhost:8080) (unless you change the port in `.env`).

---

## All the API Endpoints (with Examples)

### 1. Health Check
**GET** `/health`

Just checks if the server is alive. Good for monitoring or if you’re paranoid.

**Example:**
```bash
curl http://localhost:8080/health
```
**Response:**
```json
{
	"success": true,
	"message": "Server is running",
	"data": { "status": "healthy" }
}
```

---

### 2. Get User Profile
**GET** `/profile?uid=USER_ID` or `/profile?username=NAME&platform=PLATFORM`

Get a player’s Ubisoft profile by their UID, or by username+platform.

**Example:**
```bash
curl "http://localhost:8080/profile?uid=97d52709-b1ce-446d-bd66-c0312b99f0b4"
```
or
```bash
curl "http://localhost:8080/profile?username=CoolPlayer&platform=uplay"
```

**Response:**
```json
{
	"success": true,
	"message": "Profile retrieved successfully",
	"data": {
		"IdOnPlatform": "...",
		"NameOnPlatform": "...",
		"PlatformType": "pc",
		"ProfileId": "...",
		"UserId": "..."
	}
}
```

---

### 3. Get Game Stats
**GET** `/stats?gameId=GAME_ID&platform=PLATFORM&uids=UID1,UID2,...`

Get stats for one or more players. You can ask for up to 20 at once; if you send more, it’ll batch them for you.

**Example:**
```bash
curl "http://localhost:8080/stats?gameId=60859c37-949d-49e2-8fc8-6d8dc40f1a9e&platform=uplay&uids=uid1,uid2"
```

**Response:**
```json
{
	"success": true,
	"message": "Stats retrieved successfully",
	"data": [
		{
			"profileId": "...",
			"cached": false,
			"stats": { "pvp_kills": 150, "pvp_deaths": 45 }
		}
	]
}
```

---

### 4. Report a Player
**POST** `/report`

Send a report about a player (cheating, toxicity, etc). You’ll need to send a JSON body with the details. (Check the API for the exact payload.)

**Example:**
```bash
curl -X POST http://localhost:8080/report -H "Content-Type: application/json" -d '{ "reportedUserId": "...", "reason": "cheating" }'
```

---

### 5. Get Games List
**GET** `/games?platform=PLATFORM`

Supposed to list available games for a platform. (Not implemented yet, so don’t expect much!)

---


### 6. Get Stat Card
**GET** `/statscard?uid=UID&spaceId=GAME_ID`

Get a stat card for a player in a specific game. This gives you a summary of their stats in a nice, compact format.

**Example:**
```bash
curl "http://localhost:8080/statscard?uid=some-uid&spaceId=some-game-id"
```

**Response:**
```json
{
	"success": true,
	"message": "Statscard retrieved successfully",
	"data": {
		"statName": "value",
		"statName2": "value2"
	}
}
```

---

### 7. Check Username Availability
**GET** `/username/check?username=NAME&platform=PLATFORM`

Supposed to check if a username is available. (Not implemented yet either.)

---

## Error Responses

If something goes wrong, you’ll get:
```json
{
	"success": false,
	"error": "What went wrong"
}
```

---

## Environment Variables (aka .env)

See `env_sample` for all the options. The important ones:

- `APP_NAME` — Name for your app (shows in logs)
- `API_PORT` — What port to run on
- `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASS` — Redis settings (if you want caching)
- `UBISOFT_ACCOUNTS` — Your Ubisoft login(s) as a JSON array
- `CHROMIUM_CMD` — Path to Chromium (for headless login, if needed)

---

## Dev Stuff

- **Go backend**: Main code is in `main.go`, with logic in `api/` and `internal/`
- **Python client**: See `python/UbiGo.py` for a ready-to-go API client
- **Docker**: See `Dockerfile` and `docker-compose.yml`

---

## License

MIT License. Use it, break it, fix it, just don’t blame me if Ubisoft changes their API!

---

## Development

- **Go Backend**: Main entrypoint is [`main.go`](main.go). Core logic in [`api/`](api/) and [`internal/`](internal/).
- **Python Client**: See [`python/UbiGo.py`](python/UbiGo.py) for a ready-to-use API client.
- **Docker**: See [`Dockerfile`](Dockerfile) and [`docker-compose.yml`](docker-compose.yml).

---

## License

MIT License. See [LICENSE](LICENSE) if present.
