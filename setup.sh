#!/bin/bash

gomplate --context 'secrets=.devcontainer/secrets.json' -f .devcontainer/.env.tmpl -o .devcontainer/.env
gomplate --context 'data=.devcontainer/secrets.json' -f manifest.yaml.tmpl -o manifest.yaml
echo "GO RUN THIS: ngrok http --url=$(cat .devcontainer/secrets.json | jq .APP_URL) 8080"
