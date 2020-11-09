#!/usr/bin/env bash

rm -f src/*
cd src

curl -L -o pipeline.yaml https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml

curl -L -o dashboard.yaml https://github.com/tektoncd/dashboard/releases/latest/download/tekton-dashboard-release.yaml

curl -L -o trigger.yaml https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml
