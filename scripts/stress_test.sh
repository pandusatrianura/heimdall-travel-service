#!/bin/bash
# Heimdall Stress Test Utility using 'ab' (Apache Benchmark)

# Usage: ./scripts/stress_test.sh [host] [concurrency] [total_requests]
# Default: ./scripts/stress_test.sh http://localhost:8080 10 100

HOST=${1:-"http://localhost:8080"}
ENDPOINT="/api/v1/search"
CONCURRENCY=${2:-10}
REQUESTS=${3:-100}

# Define root of repo for file paths
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PAYLOAD_FILE="$SCRIPT_DIR/search_payload.json"

# Generate payload if missing
if [ ! -f "$PAYLOAD_FILE" ]; then
    echo "Creating search payload for benchmark..."
    cat <<EOF > "$PAYLOAD_FILE"
{
  "origin": "CGK",
  "destination": "DPS",
  "departureDate": "2025-12-15",
  "passengers": 1
}
EOF
fi

echo "--------------------------------------------------"
echo "Heimdall Load Test"
echo "Target: $HOST$ENDPOINT"
echo "Concurrency: $CONCURRENCY"
echo "Total Requests: $REQUESTS"
echo "--------------------------------------------------"

# Run Apache Benchmark
# -n: number of requests
# -c: concurrent requests
# -p: file containing data to POST
# -T: content-type header
ab -n $REQUESTS -c $CONCURRENCY -p "$PAYLOAD_FILE" -T "application/json" "$HOST$ENDPOINT"
