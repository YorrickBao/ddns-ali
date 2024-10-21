#!/bin/bash

GOOS=linux GOARCH=amd64 go build -o ddns-ali main.go
