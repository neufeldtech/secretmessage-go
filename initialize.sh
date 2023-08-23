#!/bin/bash

which gomplate || brew install gomplate

gomplate --context 'secrets=.devcontainer/secrets.json' -f .devcontainer/.env.tmpl -o .devcontainer/.env

