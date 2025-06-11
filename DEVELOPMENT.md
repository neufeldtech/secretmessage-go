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
