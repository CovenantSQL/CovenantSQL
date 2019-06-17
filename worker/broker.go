/*
 * Copyright 2019 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package worker

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/CovenantSQL/CovenantSQL/conf"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gogf/gf/g/container/gqueue"
)

const (
	//Publish API
	Newest  = "newest"
	DSNList = "dsnlist"

	//Subscribe API
	Write  = "write"
	Replay = "replay"
	Create = "create"
)

var (
	minerName     = "miner_test"
	publishPrefix = "/cql/miner/"
	listenPrefix  = "/cql/client/"
)

type SubscribeEvent struct {
	ClientID string
	DSN      string
	ApiName  string
	Payload  BrokerPayload
}

type BrokerPayload struct {
	BlockID        uint           `json:"block_id"`
	BlockIndex     uint           `json:"block_index"`
	ClientID       string         `json:"client_id"`
	ClientSequence uint           `json:"client_seq"`
	Events         []PayloadEvent `json:"events"`

	//Replay API
	BlockStart uint `json:"block_start"`
	IndexStart uint `json:"index_start"`
	BlockEnd   uint `json:"block_end"`
	IndexEnd   uint `json:"index_end"`
}

type PayloadEvent struct {
	Query string `json:"query_string"`
	Args  []struct {
		Name  string      `json:"name"`
		Value interface{} `json:"value"`
	} `json:"args"`
}

type MQTTClient struct {
	mqtt.Client
	ListenTopic        string
	PublishTopicPrefix string

	subscribeEventQueue *gqueue.Queue
}

func NewMQTTClient(config *conf.MQTTBrokerInfo) (c *MQTTClient) {
	if config == nil {
		return
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.Addr)
	opts.SetUsername(config.User)
	opts.SetPassword(config.Password)
	opts.SetClientID(config.User)
	opts.SetOrderMatters(true)

	c = &MQTTClient{
		Client:              mqtt.NewClient(opts),
		ListenTopic:         listenPrefix + "#",
		PublishTopicPrefix:  publishPrefix + minerName + "/",
		subscribeEventQueue: gqueue.New(),
	}
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		//TODO add log log.Error
		fmt.Printf("Connect broker failed: %v", token.Error())
		return
	}

	go c.subscribeEventLoop()

	c.Subscribe(c.ListenTopic, 1, subscribeCallback(c.subscribeEventQueue))

	return
}

func decodeTopicAPI(topic string) (clientID, dsn, apiName string) {
	args := strings.Split(strings.TrimPrefix(topic, listenPrefix), "/")
	if len(args) == 0 || len(args) > 3 {
		return
	}
	switch len(args) {
	case 1:
		clientID = args[0]
	case 2:
		clientID = args[0]
		if args[1] == Create {
			apiName = Create
		} else {
			dsn = args[1]
		}
	case 3:
		clientID = args[0]
		dsn = args[1]
		apiName = args[2]
	default:
		return
	}

	// valid check
	if apiName != Write && apiName != Replay && apiName != Create {
		apiName = ""
	}
	return
}

func subscribeCallback(eventQueue *gqueue.Queue) func(client mqtt.Client, msg mqtt.Message) {
	return func(client mqtt.Client, msg mqtt.Message) {
		fmt.Printf("TOPIC: %s\n", msg.Topic())
		clientID, dsn, apiName := decodeTopicAPI(msg.Topic())
		if apiName == "" {
			//TODO log topic, payload and error
			fmt.Printf("TOPIC: %s\n", msg.Topic())
			fmt.Printf("Invalid Topic: %s\n", msg.Payload())
			return
		}

		var payload BrokerPayload
		err := json.Unmarshal(msg.Payload(), &payload)
		if err != nil {
			//TODO log error
			fmt.Printf("Invalid MSG: %s, err: %v\n", msg.Payload(), err)
			return
		}
		fmt.Printf("Payload: %v\n", payload)
		eventQueue.Push(&SubscribeEvent{
			ApiName:  apiName,
			ClientID: clientID,
			DSN:      dsn,
			Payload:  payload,
		})
	}
}

func (c *MQTTClient) subscribeEventLoop() {
	for raw := range c.subscribeEventQueue.C {
		subscribeEvent := raw.(*SubscribeEvent)
		switch subscribeEvent.ApiName {
		case Write:
			c.processWriteEvent(subscribeEvent)
		case Replay:
			c.processReplayEvent(subscribeEvent)
		case Create:
			c.processCreateEvent(subscribeEvent)
		default:
			//TODO log error
			fmt.Printf("Unknow API name %s\n", subscribeEvent.ApiName)
		}
	}
}

// TODO
// 1. conf set for trigger mqtt logic
// 2. log
// 3. reconsider context
func (c *MQTTClient) processWriteEvent(event *SubscribeEvent) {
	fmt.Printf("Processed write event: %s %s %s\n", event.ClientID, event.DSN, event.ApiName)
	// TODO
	// 1. add to sqlchain
	// 2. publish to broker(in sqlchain query func, for other non-mqtt client data)
	// 3. make it unblock

	//c.PublishDSN(Newest, event.DSN, event.Payload, event.ClientID)
}

func (c *MQTTClient) processReplayEvent(event *SubscribeEvent) {
	fmt.Printf("Processed replay event: %s %s %s\n", event.ClientID, event.DSN, event.ApiName)
	// TODO
	// 1. find local db bin log
	// 2. publish to broker (in this func)
	// 3. make it unblock
	// 4. add a buffer for bin log to large

	//allPayload := fakedb.all(event.DSN)
	//for _, payload := range allPayload {
	//	c.PublishDSN(Replay, event.DSN, payload, event.ClientID)
	//}
}

func (c *MQTTClient) processCreateEvent(event *SubscribeEvent) {
	fmt.Printf("Create API does not support yet.\n")
}

func (c *MQTTClient) PublishDSN(apiName, dsn string, payload BrokerPayload, requestClient string) error {
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	var topic string

	switch apiName {
	case Newest:
		topic = c.PublishTopicPrefix + dsn + "/" + apiName
	case Replay:
		topic = c.PublishTopicPrefix + dsn + "/" + apiName + "/" + requestClient
	default:
		return errors.New("Invalid miner push api name" + apiName)
	}

	token := c.Publish(topic, 1, true, jsonBytes)
	if !token.Wait() {
		return token.Error()
	}
	return nil
}

func (c *MQTTClient) Close() {
	c.Unsubscribe(c.ListenTopic).Wait()
	c.subscribeEventQueue.Close()
	c.Disconnect(250)
}
