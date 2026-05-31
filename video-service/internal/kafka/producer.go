package kafka

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/IBM/sarama"

	"short-video-platform/pkg/events"
)

type Producer struct {
	producer sarama.SyncProducer
	topic    string
}

func NewProducer() (*Producer, error) {
	brokers := strings.Split(getenv("KAFKA_BROKERS", "127.0.0.1:9092"), ",")
	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	cfg.Producer.RequiredAcks = sarama.WaitForAll

	p, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return nil, fmt.Errorf("kafka producer: %w", err)
	}
	return &Producer{
		producer: p,
		topic:    getenv("KAFKA_TRANSCODE_TOPIC", events.TopicVideoTranscode),
	}, nil
}

func (p *Producer) PublishTranscode(task events.TranscodeTask) error {
	body, err := json.Marshal(task)
	if err != nil {
		return err
	}
	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(task.VideoID),
		Value: sarama.ByteEncoder(body),
	}
	_, _, err = p.producer.SendMessage(msg)
	return err
}

func (p *Producer) Close() error {
	return p.producer.Close()
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
