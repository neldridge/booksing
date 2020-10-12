package main

import (
	"encoding/json"
	"fmt"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofrs/uuid"
)

func newEvent(src, ty string, data map[string]string) (cloudevents.Event, error) {
	e := cloudevents.NewEvent()
	e.SetSource(src)
	e.SetType(ty)
	e.SetID(uuid.Must(uuid.NewV4()).String())
	e.SetTime(time.Now())
	err := e.SetData(cloudevents.ApplicationJSON, data)
	return e, err
}

func (app *booksingApp) pushEvent(e cloudevents.Event) error {

	bytes, err := json.Marshal(e)
	if err != nil {
		return err
	}

	topic := fmt.Sprintf("%s/%s/booksing", app.cfg.MQTTTopic, e.Type())

	if token := app.mqttClient.Publish(topic, 0, false, bytes); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}

func newMQTTClient(addr, id string) (mqtt.Client, error) {

	opts := mqtt.NewClientOptions().AddBroker(addr).SetClientID(id)
	opts.SetKeepAlive(2 * time.Second)
	opts.SetPingTimeout(2 * time.Second)

	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return c, nil
}
