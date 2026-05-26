#!/bin/bash

# Source environment file and export all variables to subprocesses
ENVFILE=.env
if [ -f "$ENVFILE" ]; then
  echo "Sourcing environment from $ENVFILE"
  set -a
  . "$ENVFILE"
  set +a
else
  echo "ERROR: $ENVFILE not found. Please run ./setup.sh first."
  exit 1
fi

echo "-----------------------------------------------"
echo "manifest.yaml:"
cat manifest.yaml
echo "-----------------------------------------------"

# Start the Go devrunner
echo "Starting ngrok and air concurrently using Go devrunner..."
go run devrunner/devrunner.go
