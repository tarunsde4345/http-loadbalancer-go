#!/bin/bash

echo "starting mock backends..."

PORT=9001 DELAY=50ms go run test/mockserver/main.go &
PORT=9002 DELAY=50ms go run test/mockserver/main.go &
PORT=9003 DELAY=80ms go run test/mockserver/main.go &

echo "backends started"
echo "  9001 — fast   (0ms)"
echo "  9002 — medium (50ms)"
echo "  9003 — slow   (80ms)"
echo ""
echo "waiting for backends to warm up..."

sleep 15 

echo "starting load test against http://localhost"
echo "  base traffic : 30 RPS"
echo "  spike traffic: 60 RPS"
echo "  jitter       : +/- 2 RPS"
echo "  spike window : 5s every 30s"
echo "  duration     : 2m"
echo ""


go run test/loadtest/main.go \
  -target http://localhost \
  -duration 2m \
  -base-rps 30 \
  -spike-rps 60 \
  -jitter 5 \
  -spike-every 30s \
  -spike-duration 5s

# wait keeps the script alive so ctrl+c kills all three
wait
