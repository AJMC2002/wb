# L0 Task: Demo service

This is the first task of WB's golang course.

# Running

In order to run the container, setup your .env

```bash
cp .env.example .env
nano .env # or vim .env
```

Then run the container

```bash
docker compose up -d --build
```

# Kafka

Run the `scripts/mock_produce.py` file to generate 100 random values to the message broker. Or just run the shell script `scripts/produce.sh` to push the example order.