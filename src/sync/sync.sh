#!/usr/bin/env bash

sync_image gcr.io/kaniko-project/executor:v1.3.0 registry.cn-huhehaote.aliyuncs.com/feng-566/kaniko-executor:v1.3.0

sync_image golang:1.14-alpine registry.cn-huhehaote.aliyuncs.com/feng-566/golang:1.14-alpine

sync_image alpine:3.11 registry.cn-huhehaote.aliyuncs.com/feng-566/alpine:3.11
