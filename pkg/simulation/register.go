// Copyright 2020-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package simulation

import (
	"encoding/json"
	"errors"
	kafka "github.com/Shopify/sarama"
	"net/url"
)

// Register records simulation traces
type Register interface {
	// Trace records the given object as a trace
	Trace(value interface{}) error

	// close closes the register
	close()
}

// newRegister returns a new register for the given URI
func newRegister(simulator, address string) (Register, error) {
	uri, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	switch uri.Scheme {
	case "kafka":
		topic := uri.Path
		if topic[0] == '/' {
			topic = topic[1:]
		}
		return newKafkaRegister(topic, simulator, uri.Host)
	default:
		return nil, errors.New("unknown register URI")
	}
}

// newKafkaRegister returns a new register that writes traces to Kafka
func newKafkaRegister(topic, key, address string) (Register, error) {
	config := kafka.NewConfig()
	config.Producer.Return.Successes = true
	producer, err := kafka.NewSyncProducer([]string{address}, config)
	if err != nil {
		return nil, err
	}
	return &kafkaRegister{
		topic:    topic,
		key:      key,
		producer: producer,
	}, nil
}

// kafkaRegister is a register that writes traces to Kafka
type kafkaRegister struct {
	topic    string
	key      string
	producer kafka.SyncProducer
}

func (r *kafkaRegister) Trace(value interface{}) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, _, err = r.producer.SendMessage(&kafka.ProducerMessage{
		Topic: r.topic,
		Key:   kafka.StringEncoder(r.key),
		Value: kafka.ByteEncoder(bytes),
	})
	return err
}

func (r *kafkaRegister) close() {
	r.producer.Close()
}
