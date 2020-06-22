// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Sample pubsub demonstrates use of the cloud.google.com/go/pubsub package from App Engine flexible environment.
package main

import (
	"context"
	"encoding/json"

	"cloud.google.com/go/pubsub"
	"github.com/gnur/booksing"
)

func (app *booksingApp) publishConvertJob(r booksing.ConvertRequest) error {
	ctx := context.Background()

	client, err := pubsub.NewClient(ctx, app.cfg.Project)
	if err != nil {
		return err
	}

	topic := client.Topic(app.cfg.TopicName)

	// Create the topic if it doesn't exist.
	exists, err := topic.Exists(ctx)
	if err != nil {
		return err
	}
	if !exists {
		_, err = client.CreateTopic(ctx, app.cfg.TopicName)
		if err != nil {
			return err
		}
	}

	raw, err := json.Marshal(r)
	if err != nil {
		return err
	}

	msg := &pubsub.Message{
		Data: raw,
	}

	if _, err := topic.Publish(ctx, msg).Get(ctx); err != nil {
		return err
	}
	return nil

}
