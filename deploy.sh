#!/bin/bash

git_status=$(git status --porcelain=v1 2>/dev/null | wc -l)
if [ $git_status -eq 0 ]; then
    ./do-deploy.sh > deploy-logs.txt 2>&1 &
else
    echo "Commit git changes first."
fi
