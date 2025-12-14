import argparse
import json
import os
import random
import time
import uuid
from copy import deepcopy
from datetime import datetime, timezone

from kafka import KafkaProducer


def mutate_order(base: dict) -> dict:
    o = deepcopy(base)
    uid = uuid.uuid4().hex
    o["order_uid"] = uid
    o["track_number"] = f"TRACK{random.randint(100000,999999)}"
    o["customer_id"] = f"cust{random.randint(1,500)}"
    o["date_created"] = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

    # payment transaction usually matches order_uid in your validation
    o["payment"]["transaction"] = uid
    o["payment"]["amount"] = random.randint(100, 50000)
    o["payment"]["payment_dt"] = int(time.time())

    # items
    for it in o.get("items", []):
        it["rid"] = uuid.uuid4().hex
        it["chrt_id"] = random.randint(1_000_000, 9_999_999)
        it["price"] = random.randint(10, 5000)
        it["sale"] = random.choice([0, 10, 20, 30, 40])
        it["total_price"] = max(1, int(it["price"] * (100 - it["sale"]) / 100))
        it["nm_id"] = random.randint(1_000_000, 9_999_999)
        it["status"] = random.choice([200, 201, 202, 203])

    return o


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--bootstrap", default=os.getenv("BOOTSTRAP", "localhost:9092"))
    ap.add_argument("--topic", default=os.getenv("TOPIC", "orders"))
    ap.add_argument("--count", type=int, default=100)
    ap.add_argument("--delay-ms", type=int, default=0)
    ap.add_argument("--template", default=os.getenv("TEMPLATE", "scripts/sample_order.json"))
    args = ap.parse_args()

    with open(args.template, "r", encoding="utf-8") as f:
        base = json.load(f)

    producer = KafkaProducer(
        bootstrap_servers=args.bootstrap,
        value_serializer=lambda v: json.dumps(v, separators=(",", ":")).encode("utf-8"),
        acks="all",
        retries=5,
        linger_ms=10,
    )

    for i in range(args.count):
        msg = mutate_order(base)
        producer.send(args.topic, value=msg)
        if (i + 1) % 25 == 0:
            producer.flush()
            print(f"[mock] sent {i+1}/{args.count}")
        if args.delay_ms:
            time.sleep(args.delay_ms / 1000)

    producer.flush()
    producer.close()
    print("[mock] done")


if __name__ == "__main__":
    main()
