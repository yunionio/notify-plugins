// Copyright 2019 Yunion
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

package common

import (
	"context"

	"yunion.io/x/notify-plugin/pkg/apis"
	"yunion.io/x/pkg/errors"
)

type ISender interface {
	IsReady(ctx context.Context) bool
	CheckConfig(ctx context.Context, configs map[string]string) (interface{}, error)
	UpdateConfig(ctx context.Context, configs map[string]string) error
	ValidateConfig(ctx context.Context, configs interface{}) (bool, string, error)
	FetchContact(ctx context.Context, related string) (string, error)
	Send(ctx context.Context, params *apis.SendParams) error
}

type SSenderBase struct {
	ConfigCache *SConfigCache
	workerChan  chan struct{}
}

func (self *SSenderBase) Do(f func() error) error {
	self.workerChan <- struct{}{}
	defer func() {
		<-self.workerChan
	}()
	return f()
}

func (self *SSenderBase) IsReady(ctx context.Context) bool {
	return true
}

func (self *SSenderBase) CheckConfig(ctx context.Context, configs map[string]string) (interface{}, error) {
	return nil, errors.ErrNotImplemented
}

func (self *SSenderBase) UpdateConfig(ctx context.Context, configs map[string]string) error {
	return errors.ErrNotImplemented
}

func (self *SSenderBase) ValidateConfig(ctx context.Context, configs interface{}) (bool, string, error) {
	return false, "", errors.ErrNotImplemented
}

func (self *SSenderBase) FetchContact(ctx context.Context, related string) (string, error) {
	return "", errors.ErrNotImplemented
}

func (self *SSenderBase) Send(ctx context.Context, params *apis.SendParams) error {
	return errors.ErrNotImplemented
}

func NewSSednerBase(config IServiceOptions) SSenderBase {
	return SSenderBase{
		ConfigCache: NewConfigCache(),
		workerChan:  make(chan struct{}, config.GetSenderNum()),
	}
}
