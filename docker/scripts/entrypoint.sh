#!/bin/bash
cd /app || exit 1
go install -v ./cmd/courier && chmod +x ./courier || exit 1
./courier
