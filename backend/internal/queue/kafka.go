package queue

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"imageprocessor/backend/internal/models"

	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(brokers []string, topic string) *KafkaProducer {
	return &KafkaProducer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.Hash{},
		},
	}
}

func (p *KafkaProducer) Publish(ctx context.Context, message models.ProcessMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	kafkaMessage := kafka.Message{
		Key:   []byte(message.ID),
		Value: payload,
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if err := p.writer.WriteMessages(ctx, kafkaMessage); err != nil {
			lastErr = err
			if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return ctx.Err()
			}
			time.Sleep(time.Duration(attempt+1) * 300 * time.Millisecond)
			continue
		}
		return nil
	}
	return lastErr
}

func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}

type KafkaConsumer struct {
	reader *kafka.Reader
}

func NewKafkaConsumer(brokers []string, topic string, groupID string) *KafkaConsumer {
	return &KafkaConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		}),
	}
}

func (c *KafkaConsumer) Consume(ctx context.Context, handler Handler) error {
	for {
		message, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("fetch kafka message: %v", err)
			time.Sleep(time.Second)
			continue
		}

		var processMessage models.ProcessMessage
		if err := json.Unmarshal(message.Value, &processMessage); err != nil {
			log.Printf("skip invalid kafka message: %v", err)
			_ = c.reader.CommitMessages(ctx, message)
			continue
		}

		if err := handler(ctx, processMessage); err != nil {
			log.Printf("process image %s: %v", processMessage.ID, err)
			_ = c.reader.CommitMessages(ctx, message)
			continue
		}
		if err := c.reader.CommitMessages(ctx, message); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("commit kafka message: %v", err)
		}
	}
}

func (c *KafkaConsumer) Close() error {
	return c.reader.Close()
}
