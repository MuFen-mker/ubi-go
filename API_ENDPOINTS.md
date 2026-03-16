# API Endpoints Documentation

## Base URL
`http://localhost:8080`

## Endpoints

### Health Check
**GET** `/health`

Check if the server is running.

**Response:**
```json
{
  "success": true,
  "message": "Server is running",
  "data": {
    "status": "healthy"
  }
}
```

---

### Get User Profile
**GET** `/profile?uid={uid}`

Retrieve a user profile by UID.

**Parameters:**
- `uid` (required): User ID

**Example:**
```bash
curl "http://localhost:8080/profile?uid=97d52709-b1ce-446d-bd66-c0312b99f0b4"
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

### Get Game Stats
**GET** `/stats?gameId={gameId}&platform={platform}&uids={uid1,uid2,...}`

Retrieve game statistics for one or more player IDs. Supports up to 20 UIDs per request; batches larger requests automatically.

**Parameters:**
- `gameId` (required): Game space ID (e.g., `60859c37-949d-49e2-8fc8-6d8dc40f1a9e` for Division 2)
- `platform` (required): Platform type (e.g., `uplay`)
- `uids` (required): Comma-separated list of player IDs

**Example:**
```bash
curl "http://localhost:8080/stats?gameId=60859c37-949d-49e2-8fc8-6d8dc40f1a9e&platform=uplay&uids=168b587a-4518-4d6a-b29f-7c8dde3e8862,e3c24627-230b-4e74-96bd-20f6106ede9b"
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
      "stats": {
        "pvp_kills": 150,
        "pvp_deaths": 45,
        ...
      }
    },
    ...
  ]
}
```

---

### Get Games List
**GET** `/games?platform={platform}`

Retrieve available games for a platform.

**Parameters:**
- `platform` (required): Platform type (e.g., `uplay`)

**Status:** ⚠️ Not yet implemented

---

### Get Stat Cards
**GET** `/statcards?gameId={gameId}&uids={uid1,uid2,...}`

Retrieve stat cards for one or more player IDs.

**Status:** ⚠️ Not yet implemented

---

### Check Username Availability
**GET** `/username/check?username={username}&platform={platform}`

Check if a username is available on a specified platform.

**Status:** ⚠️ Not yet implemented

---

## Error Responses

All error responses follow this format:

```json
{
  "success": false,
  "error": "Error message describing what went wrong"
}
```

**Common HTTP Status Codes:**
- `200 OK` - Successful request
- `400 Bad Request` - Missing or invalid parameters
- `501 Not Implemented` - Endpoint not yet implemented
- `500 Internal Server Error` - Server error

---

## Features

✅ **Automatic Batching**: Stats requests with >20 UIDs are automatically split into batches  
✅ **Response Caching**: Stats are cached for 5 minutes when Redis is available  
✅ **CORS Support**: Cross-origin requests are enabled  
✅ **Request Logging**: All requests are logged with timing information  
✅ **Error Handling**: Consistent error responses with helpful messages
