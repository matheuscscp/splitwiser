#!/bin/bash

set -e

git_status=$(git status --porcelain=v1 2>/dev/null | wc -l)
if [ $git_status -eq 0 ]; then
    ./do-deploy.sh > deploy-logs.txt 2>&1 &
else
    echo "Are you sure? You have uncommitted changes, consider using scripts/update-branch.sh."
    exit 1
fi

branch=$(git branch --show-current)
git fall
git update-ref refs/heads/main origin/main
git checkout main
git branch -D $branch
