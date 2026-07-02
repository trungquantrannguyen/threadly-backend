package messaging

import (
	"context"
	"encoding/json"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/config"
)

const (
	ExchangeName = "threadly.events"

	EventPostCreated    = "post.created"
	EventPostDeleted    = "post.deleted"
	EventUserFollowed   = "user.followed"
	EventUserUnfollowed = "user.unfollowed"

	EventPostLiked    = "post.liked"
	EventReplyCreated = "reply.created"
	EventPostReposted = "post.reposted"
	EventPostUpdated  = "post.updated"

	MaxRetries  = int32(3)
	RetryHeader = "x-retry-count"

	OriginalRoutingKeyHeader = "x-original-routing-key"
	FailureErrorHeader       = "x-failure-error"
	FailedAtHeader           = "x-failed-at"
)

type Event struct {
	EventID      string `json:"event_id"`
	Type         string `json:"type"`
	ActorID      string `json:"actor_id,omitempty"`
	PostID       string `json:"post_id,omitempty"`
	AuthorID     string `json:"author_id,omitempty"`
	TargetUserID string `json:"target_user_id,omitempty"`
	CreatedAt    string `json:"created_at"`
}

type Publisher interface {
	Publish(ctx context.Context, routingKey string, payload any) error
}

type EventHandler func(ctx context.Context, event Event) error

type RabbitMQ struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
	log      zerolog.Logger
}

func NewRabbitMQ(cfg config.Config, log zerolog.Logger) (*RabbitMQ, error) {
	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	r := &RabbitMQ{
		conn:     conn,
		channel:  ch,
		exchange: ExchangeName,
		log:      log,
	}

	if err := r.declareExchange(); err != nil {
		_ = r.Close()
		return nil, err
	}

	return r, nil
}

func (r *RabbitMQ) declareExchange() error {
	return r.channel.ExchangeDeclare(
		r.exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
}

func (r *RabbitMQ) Publish(ctx context.Context, routingKey string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return r.channel.PublishWithContext(
		ctx,
		r.exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now().UTC(),
			Body:         body,
		},
	)
}

func (r *RabbitMQ) publishRaw(
	ctx context.Context,
	routingKey string,
	body []byte,
	headers amqp.Table,
) error {
	return r.channel.PublishWithContext(
		ctx,
		r.exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now().UTC(),
			Headers:      headers,
			Body:         body,
		},
	)
}

func retryCount(headers amqp.Table) int32 {
	if headers == nil {
		return 0
	}

	value, ok := headers[RetryHeader]
	if !ok {
		return 0
	}

	switch v := value.(type) {
	case int32:
		return v
	case int64:
		return int32(v)
	case int:
		return int32(v)
	case float64:
		return int32(v)
	default:
		return 0
	}
}

func copyHeaders(headers amqp.Table) amqp.Table {
	copied := amqp.Table{}

	for key, value := range headers {
		copied[key] = value
	}

	return copied
}

func dlqRoutingKey(queueName string) string {
	return queueName + ".dlq"
}

func (r *RabbitMQ) Consume(
	ctx context.Context,
	queueName string,
	bindingKeys []string,
	handler EventHandler,
) error {
	queue, err := r.channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	dlqName := dlqRoutingKey(queueName)

	dlq, err := r.channel.QueueDeclare(
		dlqName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	if err := r.channel.QueueBind(
		dlq.Name,
		dlqName,
		r.exchange,
		false,
		nil,
	); err != nil {
		return err
	}

	for _, key := range bindingKeys {
		if err := r.channel.QueueBind(
			queue.Name,
			key,
			r.exchange,
			false,
			nil,
		); err != nil {
			return err
		}
	}

	messages, err := r.channel.Consume(
		queue.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case msg, ok := <-messages:
				if !ok {
					return
				}

				var event Event
				if err := json.Unmarshal(msg.Body, &event); err != nil {
					r.log.Error().Err(err).Msg("failed to unmarshal rabbitmq event")
					_ = msg.Nack(false, false)
					continue
				}

				if err := handler(ctx, event); err != nil {
					currentRetryCount := retryCount(msg.Headers)

					r.log.Error().
						Err(err).
						Str("event_type", event.Type).
						Int32("retry_count", currentRetryCount).
						Msg("failed to handle rabbitmq event")

					headers := copyHeaders(msg.Headers)

					if currentRetryCount < MaxRetries {
						headers[RetryHeader] = currentRetryCount + 1

						if publishErr := r.publishRaw(ctx, msg.RoutingKey, msg.Body, headers); publishErr != nil {
							r.log.Error().
								Err(publishErr).
								Str("event_type", event.Type).
								Msg("failed to republish rabbitmq retry message")

							_ = msg.Nack(false, true)
							continue
						}

						_ = msg.Ack(false)
						continue
					}

					headers[RetryHeader] = currentRetryCount
					headers[OriginalRoutingKeyHeader] = msg.RoutingKey
					headers[FailureErrorHeader] = err.Error()
					headers[FailedAtHeader] = time.Now().UTC().Format(time.RFC3339)

					if publishErr := r.publishRaw(ctx, dlqRoutingKey(queueName), msg.Body, headers); publishErr != nil {
						r.log.Error().
							Err(publishErr).
							Str("event_type", event.Type).
							Msg("failed to publish rabbitmq message to dlq")

						_ = msg.Nack(false, true)
						continue
					}

					r.log.Error().
						Str("event_type", event.Type).
						Int32("retry_count", currentRetryCount).
						Str("dlq", dlqRoutingKey(queueName)).
						Msg("rabbitmq message moved to dlq")

					_ = msg.Ack(false)
					continue
				}

				_ = msg.Ack(false)
			}
		}
	}()

	return nil
}

func (r *RabbitMQ) Close() error {
	if r.channel != nil {
		_ = r.channel.Close()
	}

	if r.conn != nil {
		return r.conn.Close()
	}

	return nil
}
