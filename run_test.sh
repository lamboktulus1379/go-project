#!/bin/bash
export ENV='stage'
go test ./... -cover -coverprofile=coverage.out && go tool cover -html=coverage.out -o coverage.html && go tool cover -func=coverage.out -o cover.txt
