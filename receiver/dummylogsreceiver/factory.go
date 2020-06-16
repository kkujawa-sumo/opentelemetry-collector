// Copyright 2019, OpenTelemetry Authors
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

package dummylogsreceiver

// This file implements factory for Jaeger receiver.

import (
	"context"

	"github.com/spf13/viper"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/internal/data"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.uber.org/zap"
)

const (
	// The value of "type" key in configuration.
	typeStr = "dummylogs"
)

// Factory is the factory for Jaeger legacy receiver.
type Factory struct {
}

type dummylogsReceiver struct {
	config   *Config
	logger   *zap.Logger
	LogConsumer consumer.LogConsumer
}

// Type gets the type of the Receiver config created by this factory.
func (f *Factory) Type() configmodels.Type {
	return configmodels.Type(typeStr)
}

// CreateDefaultConfig creates the default configuration for JaegerLegacy receiver.
func (f *Factory) CreateDefaultConfig() configmodels.Receiver {
	return &Config{
		TypeVal: configmodels.Type(typeStr),
		NameVal: typeStr,
	}
}

// CustomUnmarshaler returns the custom function to handle the special settings
// used on the receiver.
func (f *Factory) CustomUnmarshaler() component.CustomUnmarshaler {
	return func(sourceViperSection *viper.Viper, intoCfg interface{}) error {
		if sourceViperSection == nil {
			// The section is empty nothing to do, using the default config.
			return nil
		}

		// Unmarshal but not exact yet so the different keys under config do not
		// trigger errors, this is needed so that the types of protocol and transport
		// are read.
		if err := sourceViperSection.Unmarshal(intoCfg); err != nil {
			return err
		}

		// Unmarshal exact to validate the config keys.
		if err := sourceViperSection.UnmarshalExact(intoCfg); err != nil {
			return err
		}

		return nil
	}
}

func (kr *dummylogsReceiver) Shutdown(context.Context) error {
	return nil
}

func (kr *dummylogsReceiver) Start(context.Context, component.Host) error {
	return nil
}

func (f *Factory) createReceiver(
	config *Config,
) (component.LogReceiver, error) {

	r := &dummylogsReceiver{
		config:   config,
	}

	return r, nil
}

func (f *Factory) CreateLogReceiver(
	ctx context.Context,
	params component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	nextConsumer consumer.LogConsumer,
) (component.LogReceiver, error) {
	rCfg := cfg.(*Config)
	receiver, _ := f.createReceiver(rCfg)
	receiver.(*dummylogsReceiver).LogConsumer = nextConsumer
	logs := data.Logs{}
	nextConsumer.ConsumeLogs(ctx, logs)
	return receiver, nil
}
