#!/bin/bash

# Start cloudflared tunnel in the background
cloudflared tunnel --url http://localhost:8080 &

# Capture the PID of the cloudflared process
cloudflared_pid=$!

# Function to monitor stdout and stderr for the required pattern
monitor_output() {
    while read -r line; do
        if [[ $line =~ https://.*\.trycloudflare\.com ]]; then
            echo "Found pattern: $line"
            local found_string="${BASH_REMATCH[0]}"
            echo "baseUrl: ${found_string}" | /bin/gomplate -d data=stdin:///foo.yaml -f manifest.yaml.tmpl -o manifest.yaml
            chmod 777 manifest.yaml
            echo "export APP_URL=${found_string}" >> .devcontainer/.env
            echo "export SLACK_CALLBACK_URL=${found_string}/auth/slack/callback" >> .devcontainer/.env
            echo "--------------"
            echo "--------------"
            echo
            echo "Your app manifest is at manifest.yaml. Copy/paste this into slack app settings at https://api.slack.com/apps"
            echo
            echo "Start the app with ./start.sh"
            echo
            echo "--------------"
            echo "--------------"
        fi
    done
}

# Attach the monitor function to the stdout and stderr of the cloudflared process
{
    monitor_output < <(exec cloudflared tunnel --url http://localhost:8080 2>&1)
} &

# Wait for the cloudflared process to finish
wait $cloudflared_pid
