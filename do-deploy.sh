#!/bin/bash

set -e

# deploy StartBot
gcloud functions deploy StartBot \
    --runtime go116 \
    --trigger-http \
    --allow-unauthenticated \
    --security-level secure-always \
    --set-secrets 'TOKEN=start-bot-token:latest'

# create git tag for StartBot
start_bot_version=$(gcloud functions describe StartBot --format json | jq .versionId -r)
start_bot_git_tag="StartBot-v$start_bot_version"
git tag $start_bot_git_tag

# deploy Bot
gcloud functions deploy Bot \
    --runtime go116 \
    --trigger-topic start-bot \
    --set-secrets '/etc/secrets/config/latest=bot-config:latest'

# create git tag for Bot
bot_version=$(gcloud functions describe Bot --format json | jq .versionId -r)
bot_git_tag="Bot-v$bot_version"
git tag $bot_git_tag

git push --tags
