#!/bin/sh

set -eu

PORT="${1:-8008}"
FORCE=0

if [ "$PORT" = "--force" ]; then
	FORCE=1
	PORT="8008"
elif [ "${2:-}" = "--force" ]; then
	FORCE=1
fi

PIDS="$(lsof -ti tcp:"$PORT" || true)"

if [ -z "$PIDS" ]; then
	echo "Port $PORT is already free."
	exit 0
fi

echo "Process(es) on port $PORT:"
for pid in $PIDS; do
	ps -p "$pid" -o pid=,comm=,args= 2>/dev/null || echo "$pid"
done

APPROVED_PIDS=""

for pid in $PIDS; do
	CMD="$(ps -p "$pid" -o args= 2>/dev/null || true)"
	if [ -z "$CMD" ]; then
		continue
	fi

	case "$CMD" in
		*heimdall-server*|*cmd/server/main.go*|*/heimdall-travel-service/*)
			APPROVED_PIDS="$APPROVED_PIDS $pid"
			;;
		*)
			if [ "$FORCE" -ne 1 ]; then
				echo "Refusing to stop non-Heimdall process on port $PORT without --force."
				echo "Run 'make kill-port-force' or 'sh ./scripts/kill_port_8008.sh --force' if you want to stop it anyway."
				exit 1
			fi
			APPROVED_PIDS="$APPROVED_PIDS $pid"
			;;
	esac
done

APPROVED_PIDS="$(echo "$APPROVED_PIDS" | xargs)"

if [ -z "$APPROVED_PIDS" ]; then
	echo "Port $PORT released successfully."
	exit 0
fi

echo "Stopping process(es) on port $PORT: $APPROVED_PIDS"
kill $APPROVED_PIDS

for _ in 1 2 3 4 5; do
	if ! lsof -ti tcp:"$PORT" >/dev/null 2>&1; then
		echo "Port $PORT released successfully."
		exit 0
	fi
	sleep 1
done

PIDS="$(lsof -ti tcp:"$PORT" || true)"

if [ -n "$PIDS" ]; then
	if [ "$FORCE" -ne 1 ]; then
		echo "Port $PORT is still busy after a graceful stop attempt."
		echo "Run the script again with --force if you want to send SIGKILL."
		exit 1
	fi
	echo "Process(es) still running on port $PORT, forcing stop: $PIDS"
	kill -9 $PIDS
fi

echo "Port $PORT released successfully."