#!/bin/bash

cur_tag=$(gcloud container images list-tags us-central1-docker.pkg.dev/splitwiser-356519/splitwiser/splitwiser --format=json | jq '.[0].tags[-1]' -r)

echo "The current tag is $cur_tag."

cur_tag_number=$(echo $cur_tag | grep -oP '\d+')
new_tag_number=$((cur_tag_number+1))
new_tag="tag$new_tag_number"

echo "The new tag is $new_tag."

new_tag_fqn="us-central1-docker.pkg.dev/splitwiser-356519/splitwiser/splitwiser:$new_tag"

docker build . -t $new_tag_fqn
docker push $new_tag_fqn

gcloud compute instances update-container splitwiser --container-image=$new_tag_fqn
