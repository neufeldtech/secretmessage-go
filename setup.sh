#!/bin/bash

gomplate --context 'secrets=.devcontainer/secrets.json' -f .devcontainer/.env.tmpl -o .devcontainer/.env
echo "GO RUN THIS: ngrok http --url=$(cat .devcontainer/secrets.json | jq .APP_URL) 8080"
