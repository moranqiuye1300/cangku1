package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"golang.org/x/sync/errgroup"

	"short-video-platform/gen/videopb"
	"short-video-platform/pkg/events"
	"short-video-platform/transcode-worker/internal/aitag"
	"short-video-platform/transcode-worker/internal/ffmpeg"
)

type Worker struct {
	videoClient videopb.VideoServiceClient
	transcoder  *ffmpeg.Transcoder
	tagClient   *aitag.Client
	limit       int
}

func New(videoClient videopb.VideoServiceClient, transcoder *ffmpeg.Transcoder, tagClient *aitag.Client, limit int) *Worker {
	if limit < 1 {
		limit = 2
	}
	return &Worker{videoClient: videoClient, transcoder: transcoder, tagClient: tagClient, limit: limit}
}

type consumerGroupHandler struct {
	worker *Worker
	sem    chan struct{}
}

func (consumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (consumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		h.sem <- struct{}{}
		err := h.worker.handleMessage(session.Context(), msg.Value)
		<-h.sem
		if err != nil {
			log.Printf("transcode failed: %v", err)
		}
		session.MarkMessage(msg, "")
	}
	return nil
}

func (w *Worker) handleMessage(ctx context.Context, payload []byte) error {
	var task events.TranscodeTask
	if err := json.Unmarshal(payload, &task); err != nil {
		return err
	}
	log.Printf("transcode start video=%s source=%s", task.VideoID, task.SourcePath)

	result, err := w.transcoder.Transcode(ctx, task.VideoID, task.SourcePath)
	if err != nil {
		_, _ = w.videoClient.UpdateTranscodeResult(ctx, &videopb.UpdateTranscodeResultRequest{
			VideoId:      task.VideoID,
			Status:       "failed",
			ErrorMessage: err.Error(),
		})
		return err
	}

	_, err = w.videoClient.UpdateTranscodeResult(ctx, &videopb.UpdateTranscodeResultRequest{
		VideoId:  task.VideoID,
		Status:   "ready",
		Duration: result.Duration,
		CoverUrl: result.CoverURL,
		PlayUrls: result.PlayURLs,
		Tags:     w.generateTags(ctx, task.Title, task.Description),
	})
	if err != nil {
		return fmt.Errorf("update transcode result: %w", err)
	}
	log.Printf("transcode done video=%s duration=%ds", task.VideoID, result.Duration)
	return nil
}

func (w *Worker) generateTags(ctx context.Context, title, description string) []string {
	if w.tagClient == nil {
		return nil
	}
	tags, err := w.tagClient.Generate(ctx, title, description)
	if err != nil {
		log.Printf("ai tag warning video title=%q: %v", title, err)
		return nil
	}
	return tags
}

func Run(ctx context.Context, w *Worker) error {
	brokers := strings.Split(getenv("KAFKA_BROKERS", "127.0.0.1:9092"), ",")
	topic := getenv("KAFKA_TRANSCODE_TOPIC", events.TopicVideoTranscode)
	group := getenv("KAFKA_CONSUMER_GROUP", "transcode-worker")

	cfg := sarama.NewConfig()
	cfg.Version = sarama.V3_6_0_0
	cfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRange()}
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest

	groupClient, err := sarama.NewConsumerGroup(brokers, group, cfg)
	if err != nil {
		return err
	}
	defer groupClient.Close()

	handler := consumerGroupHandler{
		worker: w,
		sem:    make(chan struct{}, w.limit),
	}

	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		for {
			if err := groupClient.Consume(egCtx, []string{topic}, handler); err != nil {
				return err
			}
			if egCtx.Err() != nil {
				return egCtx.Err()
			}
		}
	})
	eg.Go(func() error {
		for {
			select {
			case <-egCtx.Done():
				return egCtx.Err()
			case err := <-groupClient.Errors():
				if err != nil {
					log.Printf("kafka consumer error: %v", err)
				}
			}
		}
	})
	return eg.Wait()
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func ParseDurationEnv(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
