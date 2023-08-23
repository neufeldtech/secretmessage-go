#!/bin/bash
set -x
ENVFILE=.devcontainer/.env
ENVFILEURLS=.devcontainer/.urlenv
echo "sourcing $ENVFILE"
. $ENVFILE
echo "sourcing $ENVFILEURLS"
. $ENVFILEURLS
set +x
echo "-----------------------------------------------"
echo "-----------------------------------------------"
echo "-----------------------------------------------"
echo "-----------------------------------------------"
echo manifest.yaml
echo
cat manifest.yaml
echo
echo
echo "-----------------------------------------------"
echo "-----------------------------------------------"
echo "-----------------------------------------------"
echo "-----------------------------------------------"
air
