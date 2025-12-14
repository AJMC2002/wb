#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

TOPIC="${1:-${KAFKA_TOPIC:-orders}}"
KAFKA_SERVICE="${KAFKA_SERVICE:-kafka}"
BOOTSTRAP="${BOOTSTRAP:-kafka:9093}"
MSG_FILE="${MSG_FILE:-$ROOT_DIR/scripts/sample_order.json}"

if [[ ! -f "$MSG_FILE" ]]; then
  echo "[produce] Message file not found: $MSG_FILE" >&2
  exit 1
fi

echo "[produce] kafka service : $KAFKA_SERVICE"
echo "[produce] bootstrap     : $BOOTSTRAP"
echo "[produce] topic         : $TOPIC"
echo "[produce] msg file      : $MSG_FILE"

echo "[produce] ensuring topic exists..."
docker compose exec -T "$KAFKA_SERVICE" sh -lc "
  set -eux
  /opt/kafka/bin/kafka-topics.sh \
    --bootstrap-server '$BOOTSTRAP' \
    --create --if-not-exists \
    --topic '$TOPIC' \
    --partitions 1 \
    --replication-factor 1
"

echo "[produce] sending message..."
docker compose exec -T "$KAFKA_SERVICE" sh -lc "
  set -eux
  tr -d '\n' | /opt/kafka/bin/kafka-console-producer.sh \
    --bootstrap-server '$BOOTSTRAP' \
    --topic '$TOPIC'
" < "$MSG_FILE"

echo "[produce] done"