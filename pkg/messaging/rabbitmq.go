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
					r.log.Error().
						Err(err).
						Str("event_type", event.Type).
						Msg("failed to handle rabbitmq event")

					_ = msg.Nack(false, true)
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
