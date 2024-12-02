#!/bin/bash
cd /app || exit 1
go install -v ./cmd/courier && chmod +x /go/bin/courier || exit 1
/go/bin/courier
