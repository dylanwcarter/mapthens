# Mapthens

Local event mapping application for Athens, GA. Scrapes events from Flagpole.com and displays them on a Mapbox map.

## Prerequisites

- Go (1.21+)
- Mapbox Access Token

## Setup & Run

1. Export your Mapbox Access Token:
   ```bash
   export MAPBOX_ACCESS_TOKEN="your_pk_token_here"
   ```

2. Run the application:
   ```bash
   ./run.sh
   ```

3. Open your browser to:
   [http://localhost:8080](http://localhost:8080)

## Architecture

- **server/**: Go backend that scrapes events, stores them locally in `events.json`, and serves the API and static files.
- **public/**: Frontend assets (HTML, JS, CSS).

## Notes

- The server will scrape events on the first run and cache them in `server/events.json`.
- Subsequent runs will use the cached file unless it is deleted or the server logic is updated to invalidate it.