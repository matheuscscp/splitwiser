#!/bin/bash

set -e

key_file_suffix="service-account-key.json"

project_id=$(gcloud config list --format 'value(core.project)' 2>/dev/null)
account_name="terraform"
email="$account_name@$project_id.iam.gserviceaccount.com"
tmp_key_file="tmp-$key_file_suffix"

if [[ $(gcloud iam service-accounts list --format=json | jq '.[].email' -r | grep -P "^$account_name@") ]]; then
    echo "There's already a service account for Terraform, if you want to rotate it then please delete the old account first:"
    echo ""
    echo "gcloud projects remove-iam-policy-binding $project_id --member='serviceAccount:$email' --role='roles/owner'; \\"
    echo "    gcloud iam service-accounts delete $email"
    exit 1
fi

gcloud iam service-accounts create $account_name \
    --description 'Service account for Terraform' \
    --display-name 'Terraform'
gcloud projects add-iam-policy-binding $project_id \
    --member="serviceAccount:$email" \
    --role='roles/owner'
gcloud iam service-accounts keys create $tmp_key_file \
    --iam-account=$email

key_id=$(cat $tmp_key_file | jq '.private_key_id' -r)
key_file="$email-$key_id-$key_file_suffix"
cat $tmp_key_file | jq -r tostring > $key_file
rm $tmp_key_file

echo ""
echo "Service account key file created at $key_file."
echo "Set GOOGLE_CREDENTIALS as a secret environment variable for Terraform with the contents of this JSON file."
