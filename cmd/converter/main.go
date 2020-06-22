package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/gnur/booksing"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

type configuration struct {
	TopicName        string `required:"true"`
	SubscriptionName string `required:"true"`
	GoogleProject    string `required:"true"`
	BooksingHost     string `required:"true"`
	BooksingAPIKey   string `required:"true"`
}

func main() {

	var cfg configuration

	err := envconfig.Process("converter", &cfg)
	if err != nil {
		log.WithField("err", err).Fatal("Could not load required config")
	}
	log.SetLevel(log.DebugLevel)

	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, cfg.GoogleProject)
	if err != nil {
		log.Fatalf("Could not create pubsub Client: %v", err)
	}

	log.Debug("getting topic")
	t := getTopic(client, cfg.TopicName)

	log.Debug("creating subscription")
	if err := create(client, cfg.SubscriptionName, t); err != nil {
		log.Fatal(err)
	}

	log.Info("starting subscription listener")
	if err := cfg.handleMSG(client, cfg.SubscriptionName, t); err != nil {
		log.Fatal(err)
	}
}

func create(client *pubsub.Client, subName string, topic *pubsub.Topic) error {
	ctx := context.Background()
	sub := client.Subscription(subName)
	ok, err := sub.Exists(ctx)
	if ok {
		return nil
	}
	sub, err = client.CreateSubscription(ctx, subName, pubsub.SubscriptionConfig{
		Topic:       topic,
		AckDeadline: 240 * time.Second,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Created subscription: %v\n", sub)
	return nil
}

func delete(client *pubsub.Client, subName string) error {
	ctx := context.Background()
	sub := client.Subscription(subName)
	if err := sub.Delete(ctx); err != nil {
		return err
	}
	fmt.Println("Subscription deleted.")
	return nil
}

func getTopic(c *pubsub.Client, topic string) *pubsub.Topic {
	ctx := context.Background()

	t := c.Topic(topic)
	ok, err := t.Exists(ctx)
	if err != nil {
		log.WithField("err", err).Fatal("unable to list topic")
	}
	if ok {
		return t
	}

	t, err = c.CreateTopic(ctx, topic)
	if err != nil {
		log.WithField("err", err).Fatal("Could not create topic")
	}
	return t
}

func (cfg *configuration) handleMSG(client *pubsub.Client, subName string, topic *pubsub.Topic) error {
	ctx := context.Background()

	sub := client.Subscription(subName)
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()
	err := sub.Receive(cctx, func(ctx context.Context, msg *pubsub.Message) {
		m := booksing.ConvertRequest{}
		msg.Ack()
		err := json.Unmarshal(msg.Data, &m)
		if err != nil {
			log.WithField("err", err).Warning("could not parse pubsub body")
			return
		}
		err = convertBook(m)
		if err != nil {
			log.WithField("err", err).Warning("conversion failed")
			return
		}
		err = cfg.addToBooksing(m)
		if err != nil {
			log.WithField("err", err).Warning("could not add to booksing")
			return
		}
		log.Info("Conversion succeeded")

	})

	//this code is probably never reached
	if err != nil {
		return err
	}
	return nil
}
