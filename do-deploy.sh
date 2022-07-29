#!/bin/bash

set -e

# deploy StartBot
gcloud functions deploy StartBot \
    --runtime go116 \
    --region europe-west2 \
    --trigger-http \
    --allow-unauthenticated \
    --security-level secure-always \
    --set-secrets 'TOKEN=start-bot-token:latest' \
    --set-env-vars 'PROJECT_ID=splitwiser-356519' \
    --set-env-vars 'TOPIC_ID=start-bot'

# create git tag for StartBot
start_bot_version=$(gcloud functions describe StartBot --region europe-west2 --format json | jq .versionId -r)
start_bot_git_tag="StartBot-v$start_bot_version"
git tag $start_bot_git_tag

# deploy Bot
gcloud functions deploy Bot \
    --runtime go116 \
    --region europe-west2 \
    --trigger-topic start-bot \
    --timeout 540s \
    --set-secrets '/etc/secrets/config/latest.yml=bot-config:latest' \
    --set-env-vars 'CONF_FILE=/etc/secrets/config/latest.yml'

# create git tag for Bot
bot_version=$(gcloud functions describe Bot --region europe-west2 --format json | jq .versionId -r)
bot_git_tag="Bot-v$bot_version"
git tag $bot_git_tag

git push --tags
