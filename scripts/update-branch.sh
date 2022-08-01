#!/bin/bash

set -e

git add .
git stash
git fall
git update-ref refs/heads/main origin/main
git reset --hard main
git stash pop
