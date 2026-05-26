#!/bin/bash

# Ensure .env exists
if [ ! -f .env ]; then
  echo "Copying .env.example to .env..."
  cp .env.example .env
fi

# Ensure postgresql@14 is installed via Homebrew
if ! brew list --formula | grep -q "^postgresql@14$"; then
  echo "postgresql@14 is not installed. Installing via Homebrew..."
  brew install postgresql@14
fi

SERVICE_NAME="postgresql@14"
PG_BIN="$(brew --prefix postgresql@14)/bin"

echo "Using PostgreSQL service: $SERVICE_NAME"
echo "Using PG binaries from: $PG_BIN"

# Locate and configure port 5432 in postgresql.conf
CONF_FILES=$(find "$(brew --prefix)/var" -name postgresql.conf 2>/dev/null)
for CONF in $CONF_FILES; do
  echo "Configuring port 5432 in $CONF..."
  # Clean up any custom 5433 port configs we did earlier, setting it back to 5432
  sed -i.bak 's/^[#]*port = 5433/port = 5432/' "$CONF"
done

# Restart service to apply changes and ensure it's running
echo "Starting/Restarting $SERVICE_NAME service via Homebrew..."
brew services restart "$SERVICE_NAME"

# Wait up to 15 seconds for postgres to start on port 5432
echo "Waiting for PostgreSQL to start on port 5432..."
for i in {1..15}; do
  if "$PG_BIN/pg_isready" -p 5432 &>/dev/null; then
    echo "PostgreSQL started successfully on port 5432."
    break
  fi
  sleep 1
done

if ! "$PG_BIN/pg_isready" -p 5432 &>/dev/null; then
  echo "ERROR: PostgreSQL service did not start on port 5432. Please check brew services status."
  exit 1
fi

# Create root user if not exists
echo "Ensuring 'root' role exists on port 5432..."
"$PG_BIN/psql" -p 5432 -d postgres -c "SELECT 1 FROM pg_roles WHERE rolname='root'" | grep -q 1
if [ $? -ne 0 ]; then
  echo "Creating 'root' role with SUPERUSER privileges..."
  "$PG_BIN/psql" -p 5432 -d postgres -c "CREATE ROLE root WITH SUPERUSER LOGIN;"
else
  echo "'root' role already exists."
fi

# Create secretmessage database if not exists
echo "Ensuring 'secretmessage' database exists on port 5432..."
"$PG_BIN/psql" -p 5432 -d postgres -c "SELECT 1 FROM pg_database WHERE datname='secretmessage'" | grep -q 1
if [ $? -ne 0 ]; then
  echo "Creating database 'secretmessage' owned by 'root'..."
  "$PG_BIN/psql" -p 5432 -d postgres -c "CREATE DATABASE secretmessage OWNER root;"
else
  echo "Database 'secretmessage' already exists."
fi

# Read APP_URL from .env
echo "Generating manifest.yaml..."
APP_URL=$(grep '^export APP_URL=' .env | cut -d= -f2- || grep '^APP_URL=' .env | cut -d= -f2-)
APP_URL=$(echo "$APP_URL" | sed -e 's/^"//' -e 's/"$//' -e "s/^'//" -e "s/'$//")

if [ -z "$APP_URL" ]; then
  echo "WARNING: APP_URL not found in .env. manifest.yaml might be invalid."
else
  echo "Using APP_URL: $APP_URL"
fi

# Replace __APP_URL__ in manifest.yaml.tmpl and generate manifest.yaml
sed "s|__APP_URL__|${APP_URL}|g" manifest.yaml.tmpl > manifest.yaml

echo "------------------------------------------------------------"
echo "Setup complete!"
echo "You can now run: ./start.sh"
echo "------------------------------------------------------------"
