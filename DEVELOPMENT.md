- This app uses visual studio code devcontainers

# Running Locally
- Start the devcontainer
- Observe Cloudflare Tunnel Temp Name for this session:

```
2023-08-19T11:56:34Z INF |  Your quick Tunnel has been created! Visit it at (it may take some time to be reachable):  |
2023-08-19T11:56:34Z INF |  https://instructions-feed-focuses-considering.trycloudflare.com                           |
2023-08-19T11:56:34Z INF +--------------------------------------------------------------------------------------------+
```

- Update dev slack manifest with this name (https://app.slack.com/app-settings/foo/bar)
- Fix CALLBACK_URL and populate other env variables somehow... (source them in container?)
- Boot server `go run .`
- Reinstall app? (https://instructions-feed-focuses-considering.trycloudflare.com/auth/slack)
- Profit
