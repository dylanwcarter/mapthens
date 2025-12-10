#!/bin/bash

if [ -z "$MAPBOX_ACCESS_TOKEN" ]; then
  echo "Error: MAPBOX_ACCESS_TOKEN is not set."
  echo "Please export it: export MAPBOX_ACCESS_TOKEN=your_token_here"
  exit 1
fi

echo "Starting Mapthens server..."
cd server
go run main.go
