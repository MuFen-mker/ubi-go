# Ubi-Go - API Proxy Interface

## Overview
This document outlines the design of a Go-based Ubisoft API / Proxy. The client interacts with Ubisoft's API to retrieve game data, user profiles, game stats, and stat cards. It supports caching with Redis and a local failover, authentication credential management, and concurrent request handling. All of these are accesible through an API which should simplify data retrieval and implementations.

## Features

### Data Retrieval
- **Game List Retrieval**: Fetch a list of games with their IDs for a specified platform.
- **User Profiles**:
  - Retrieve a single user profile by UID or username.
  - Retrieve multiple user profiles by UIDs.
- **Game Statistics**:
  - Fetch stats for a single user ID.
  - Fetch stats for up to 20 user IDs in a single request.
- **Game Stat Cards**:
  - Fetch stat cards for a single user ID.
  - Fetch stat cards for up to 20 user IDs in a single request.
- **Username Availability**: Check if a username is available on a specified platform.

### Caching
- **Redis Caching**: Cache API responses and credentials in Redis with configurable TTL.
- **Local Failover**: Use a `.cache` file as a fallback if Redis is unavailable, storing tokens and responses.

### Authentication
- **Credential Storage**: Load credentials from `accounts.json` initially, then store and serve them from Redis.
- **Credential Rotation**: Rotate expired credentials and update both Redis and the local `.cache` file.
- **Failover**: Load credentials from the local `.cache` file if Redis is unavailable.

### Concurrency
- For requests exceeding 20 user IDs (stats or stat cards):
  - Use goroutines to process requests in batches of 20.
  - Ensure each batch completes within 1.5 seconds to avoid API timeouts.

## API Endpoints
- **Retrieve Games**: `/games?platform={platform}`  
  - Fetches a list of games with IDs for the specified platform.
- **Fetch Profile by Username**: `/profile?username={username}&platform={platform}`  
  - Retrieves a user profile by username and platform.
- **Fetch Profile by UID**: `/profile/{uid}?platform={platform}`  
  - Retrieves a user profile by UID and platform.
- **Fetch Game Stats**: `/stats/{gameID}?uids={uid1,uid2,...}`  
  - Fetches game statistics for up to 20 user IDs.
- **Fetch Game Stat Cards**: `/statcards/{gameID}?uids={uid1,uid2,...}`  
  - Fetches stat cards for up to 20 user IDs.
- **Username Availability Check**: `/username/check?username={username}&platform={platform}`  
  - Checks if a username is available on the specified platform.

## Implementation Details

### Data Translation
- All Ubisoft API data points (games, profiles, stats, stat cards) are translated into Go structs for type safety and ease of use.

### Caching Strategy
- **Redis**: 
  - Store responses with keys like `ubisoft:games:{platform}` or `ubisoft:stats:{gameID}:{uids}`.
  - Store credentials under `ubisoft:credentials`.
- **Local Failover**: 
  - Save responses and credentials to `.cache` if Redis fails.
  - Load from `.cache` during initialization or failover.

### Concurrency Handling
- Use goroutines and a `sync.WaitGroup` to manage batches of 20 user IDs.
- Process batches concurrently, aggregating results once all goroutines complete.