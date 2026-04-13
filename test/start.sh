#!/bin/bash

echo "starting mock backends..."

PORT=9001 DELAY=0ms go run test/mockserver/main.go &
PORT=9002 DELAY=50ms go run test/mockserver/main.go &
PORT=9003 DELAY=80ms go run test/mockserver/main.go &

echo "backends started"
echo "  9001 — fast   (0ms)"
echo "  9002 — medium (50ms)"
echo "  9003 — slow   (80ms)"

sleep 15 
go run test/loadtest/main.go

# wait keeps the script alive so ctrl+c kills all three
wait
