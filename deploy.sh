#!/bin/bash

git --no-pager diff --exit-code > /dev/null 2>&1
if [ $? -eq 0 ]; then
    ./do-deploy.sh > deploy-logs.txt 2>&1 &
else
    echo "Commit git changes first."
fi
