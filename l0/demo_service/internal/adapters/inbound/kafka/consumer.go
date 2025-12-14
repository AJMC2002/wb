package kafkain

import (
	"context"
	"errors"
	"log"
	"time"

	"demo_service/internal/core/service"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
	svc    *service.OrderService
}

type ConsumerConfig struct {
	Brokers  []string
	Topic    string
	GroupID  string
	MinBytes int
	MaxBytes int
}

func NewConsumer(cfg ConsumerConfig, svc *service.OrderService) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Brokers,
		Topic:    cfg.Topic,
		GroupID:  cfg.GroupID,
		MinBytes: cfg.MinBytes,
		MaxBytes: cfg.MaxBytes,
	})
	return &Consumer{reader: r, svc: svc}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

func (c *Consumer) Run(ctx context.Context) {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			// Normal shutdown path
			if errors.Is(err, context.Canceled) {
				return
			}
			log.Printf("[kafka] fetch error: %v", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		order, derr := DecodeOrder(msg.Value)
		if derr != nil {
			log.Printf("[kafka] bad message (skip+commit) key=%s err=%v", string(msg.Key), derr)
			_ = c.reader.CommitMessages(ctx, msg) // commit poison pill
			continue
		}

		if err := c.svc.Ingest(ctx, order); err != nil {
			// Important: do NOT commit, so message can be retried.
			log.Printf("[kafka] ingest failed (no commit) order_uid=%s err=%v", order.OrderUID, err)
			time.Sleep(1 * time.Second)
			continue
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("[kafka] commit error: %v", err)
			// commit failing means it may redeliver; acceptable for demo.
		}
	}
}
