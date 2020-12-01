package pubsub

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/rs/zerolog/log"
)

var (
	publishTimeout = 30 * time.Second
)

type Client struct {
	topic *pubsub.Topic
	sub   *pubsub.Subscription
}

func NewClient(projectID, topicName, subName string, timeout time.Duration) (*Client, error) {
	ctx := context.Background()

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	topic := client.Topic(topicName)

	// Create the topic if it doesn't exist
	exists, err := topic.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check topic existense: %v", err)
	}
	if !exists {
		if _, err = client.CreateTopic(ctx, topicName); err != nil {
			return nil, fmt.Errorf("failed to create topic")
		}
	}

	// Create subscription if it doesn't exists
	sub := client.Subscription(subName)
	exists, err = sub.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check sub existense: %v", err)
	}
	if !exists {
		if _, err = client.CreateSubscription(ctx, subName, pubsub.SubscriptionConfig{
			Topic:       topic,
			AckDeadline: timeout,
		}); err != nil {
			return nil, fmt.Errorf("failed to create sub: %v", err)
		}
	}

	return &Client{topic: topic, sub: sub}, nil
}

func (c *Client) Publish(ctx context.Context, data []byte) error {
	msg := &pubsub.Message{Data: data}

	ctx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()

	if _, err := c.topic.Publish(ctx, msg).Get(ctx); err != nil {
		return err
	}

	return nil
}

func (c *Client) Consume(ctx context.Context, handler func(ctx context.Context, data []byte) (bool, error)) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := c.sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		log.Info().Msgf("Consumed: %v", msg)
		ok, err := handler(ctx, msg.Data)
		if err != nil {
			log.Error().Err(err).Msg("Failed to handle data")
		}
		if ok {
			msg.Ack()
		} else {
			msg.Nack()
		}
	}); err != nil {
		return fmt.Errorf("receive error: %v", err)
	}
	return nil
}
