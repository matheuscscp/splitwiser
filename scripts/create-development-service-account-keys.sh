#!/bin/bash

set -e

declare -A functions=(["bot-dev"]="bot" ["start-bot-dev"]="startbot" ["rotate-secret-dev"]="rotatesecret")

for service_account in "${!functions[@]}"; do
    email=$(gcloud iam service-accounts list --format=json | jq '.[].email' -r | grep -P "^$service_account@")
    keys=$(gcloud iam service-accounts keys list --iam-account=$email --managed-by=user --format=json | jq '.[].name' -r)
    for key in $keys; do
        gcloud iam service-accounts keys delete $key --iam-account=$email
    done
    new_key_file="cmd/${functions[$service_account]}/service-account-key.json"
    gcloud iam service-accounts keys create $new_key_file --iam-account=$email
done
