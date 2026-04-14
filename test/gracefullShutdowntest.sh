#!/bin/bash

echo "starting slow backend (2s delay)..."
PORT=9001 DELAY=2s go run test/mockserver/main.go &
BACKEND_PID=$!

# wait for backend to start
sleep 2

echo "BRO! start load balancer..."

# wait for LB to compile, start, and health check to pass
echo "waiting for LB and health check..."
sleep 15

# verify backend is reachable before sending load
echo "verifying backend is alive..."
curl -s http://localhost:80/
echo ""

echo 'kill server in a second...'
sleep 1
echo "sending 50 concurrent requests (each takes 2s)..."
for i in {1..50}; do
    curl -s -w "[req $i] status=%{http_code} time=%{time_total}s\n" \
         -o /dev/null \
         http://localhost:80/ &
done

# kill while requests are mid-flight
sleep 1
kill -9 $(lsof -ti :80)

# requests take 2s, kill after 0.5s — definitely mid-flight

# wait for all curls to finish
wait

echo "cleaning up..."
kill $BACKEND_PID 2>/dev/null



# not working yet, 