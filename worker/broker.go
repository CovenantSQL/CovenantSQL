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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"os"
	"os/signal"
	"syscall"

	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
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

type fakeDB map[string][]BrokerPayload

var globalLock sync.RWMutex

func (f fakeDB) add(dsn string, payload BrokerPayload) {
	globalLock.Lock()
	defer globalLock.Unlock()
	f[dsn] = append(f[dsn], payload)
}
func (f fakeDB) last(dsn string) BrokerPayload {
	globalLock.RLock()
	defer globalLock.RUnlock()
	if len(f[dsn]) == 0 {
		return BrokerPayload{}
	}
	payload := f[dsn][len(f[dsn])-1]
	return payload
}
func (f fakeDB) all(dsn string) []BrokerPayload {
	globalLock.RLock()
	defer globalLock.RUnlock()
	var allPayload []BrokerPayload
	for _, payload := range f[dsn] {
		allPayload = append(allPayload, payload)
	}
	return allPayload
}

var fakedb fakeDB

var (
	minerName          = "miner_test"
	publishPrefix      = "/cql/miner/"
	listenPrefix       = "/cql/client/"
	subscribeEventChan chan SubscribeEvent
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
	ctx    context.Context
	cancel context.CancelFunc

	mqtt.Client
	ListenTopic        string
	PublishTopicPrefix string
}

func NewMQTTClient() (c *MQTTClient) {

	opts := mqtt.NewClientOptions()
	opts.AddBroker("192.168.2.100:18888")
	opts.SetUsername(minerName)
	opts.SetPassword("laodouya")
	opts.SetClientID(minerName)
	opts.SetOrderMatters(true)

	c = &MQTTClient{
		Client:             mqtt.NewClient(opts),
		ListenTopic:        listenPrefix + "#",
		PublishTopicPrefix: publishPrefix + minerName + "/",
	}
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		//TODO add log log.Error
		return
	}

	subscribeEventChan = make(chan SubscribeEvent)
	c.ctx, c.cancel = context.WithCancel(context.Background())
	go c.subscribeEventLoop()

	c.Subscribe(c.ListenTopic, 1, SubscribeCallback)

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

func SubscribeCallback(client mqtt.Client, msg mqtt.Message) {
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
	subscribeEventChan <- SubscribeEvent{
		ApiName:  apiName,
		ClientID: clientID,
		DSN:      dsn,
		Payload:  payload,
	}
}

func (c *MQTTClient) subscribeEventLoop() {
	for {
		select {
		// TODO remove block processing
		case subscribeEvent := <-subscribeEventChan:
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
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *MQTTClient) processWriteEvent(event SubscribeEvent) {
	fmt.Printf("Processed write event: %s %s %s\n", event.ClientID, event.DSN, event.ApiName)
	last := fakedb.last(event.DSN)
	event.Payload.BlockID = last.BlockID + 1
	fakedb.add(event.DSN, event.Payload)
	c.PublishDSN(Newest, event.DSN, event.Payload, event.ClientID)
}

func (c *MQTTClient) processReplayEvent(event SubscribeEvent) {
	fmt.Printf("Processed replay event: %s %s %s\n", event.ClientID, event.DSN, event.ApiName)
	allPayload := fakedb.all(event.DSN)
	for _, payload := range allPayload {
		c.PublishDSN(Replay, event.DSN, payload, event.ClientID)
	}
}

func (c *MQTTClient) processCreateEvent(event SubscribeEvent) {
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
	close(subscribeEventChan)
	c.cancel()
	c.Disconnect(250)
}

func main() {
	client := NewMQTTClient()
	defer client.Close()
	fakedb = make(fakeDB)
	signalCh := make(chan os.Signal, 1)
	signal.Notify(
		signalCh,
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	signal.Ignore(syscall.SIGHUP, syscall.SIGTTIN, syscall.SIGTTOU)
	<-signalCh
}
