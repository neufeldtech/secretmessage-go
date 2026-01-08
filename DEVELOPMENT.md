- This app uses visual studio code devcontainers

# Running Locally
- Populate `secrets.json` with real values
- VS Code Command Pallette: Reopen in devcontainer
- Note: Do various ngrok things
  - `brew install ngrok`
  - `ngrok config add-authtoken foobar`
  - Create permanent static domain in ngrok admin and save this to APP_URL in `.devcontainer/secrets.json`
- Update [dev slack manifest](https://api.slack.com/apps) from generated `manifest.yaml`
- Start the app `./start.sh`, you should see your ngrok tunnel domain, open that in a LOCAL terminal (not in the devcontainer)
- Reinstall app if needed (https://your-ngrok-domain.com/auth/slack)
- Profit

# Design

## Store a Secret

Slack calls POST /slash with payload containing the user's secret payload -> Generate a secretID (This becomes the key that the user will use to retrieve it later) -> We encrypt their secret using the secretID -> Hash their secretID -> store hashed secretID and encrypted secret in the db -> Send response to slack containing the 'envelope' button message containing the secretID as the callback_id

## Retrieve a Secret

User clicks the 'envelope' button in Slack -> Slack calls POST /interactive with the callback_id that was associated to this secret (which is a secretID) -> We hash the secretID to check the database for a matching secret -> Retrieve encrypted payload from database -> Decrypt payload with the secretID -> Return to user -> Delete secret from database

