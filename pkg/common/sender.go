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
	"sync"

	"yunion.io/x/notify-plugins/pkg/apis"
	"yunion.io/x/pkg/errors"
)

type ISender interface {
	IsReady(ctx context.Context) bool
	// CheckConfig(ctx context.Context, configs map[string]string) (interface{}, error)
	UpdateConfig(ctx context.Context, configs map[string]string) error
	// ValidateConfig(ctx context.Context, configs map[string]string) (bool, string, error)
	FetchContact(ctx context.Context, related string) (string, error)
	Send(ctx context.Context, params *SendParam) error
	BatchSend(ctx context.Context, params *BatchSendParam) ([]*FailedRecord, error)
}

type SenderWapper func(domainId string, senders *sync.Map) (ISender, bool)

type FailedRecord struct {
	Contact string
	Reason  string
}

type SendParam struct {
	Contact        string
	Topic          string
	Title          string
	Message        string
	Priority       string
	RemoteTemplate string
}

type BatchSendParam struct {
	Contacts       []string
	Title          string
	Message        string
	Priority       string
	RemoteTemplate string
}

type SSenderBase struct {
	ConfigCache *SConfigCache
	WorkerChan  chan struct{}
}

func (self *SSenderBase) Do(f func() error) error {
	self.WorkerChan <- struct{}{}
	defer func() {
		<-self.WorkerChan
	}()
	return f()
}

func (self *SSenderBase) IsReady(ctx context.Context) bool {
	return true
}

func (self *SSenderBase) UpdateConfig(ctx context.Context, configs map[string]string) error {
	return errors.ErrNotImplemented
}

func (self *SSenderBase) FetchContact(ctx context.Context, related string) (string, error) {
	return "", errors.ErrNotImplemented
}

func (self *SSenderBase) Send(ctx context.Context, params *apis.SendParams) error {
	return errors.ErrNotImplemented
}

func (self *SSenderBase) BatchSend(ctx context.Context, params *apis.BatchSendParams) ([]*apis.FailedRecord, error) {
	return nil, errors.ErrNotImplemented
}

func BatchSend(ctx context.Context, params *BatchSendParam, singleSend func(context.Context, *SendParam) error) ([]*FailedRecord, error) {
	ret := make([]*FailedRecord, len(params.Contacts))
	send := func(i int) {
		param := &SendParam{
			Contact:        params.Contacts[i],
			Topic:          params.Title,
			Title:          params.Title,
			Message:        params.Message,
			Priority:       params.Priority,
			RemoteTemplate: params.RemoteTemplate,
		}
		err := singleSend(ctx, param)
		if err == nil {
			return
		}
		record := &FailedRecord{
			Contact: param.Contact,
			Reason:  err.Error(),
		}
		ret[i] = record
	}
	for i := range ret {
		send(i)
	}
	// remove nil
	processedRet := make([]*FailedRecord, 0, len(ret))
	for i := range ret {
		if ret[i] == nil {
			continue
		}
		processedRet = append(processedRet, ret[i])
	}
	return processedRet, nil
}

func NewSSednerBase(config IServiceOptions) SSenderBase {
	return SSenderBase{
		ConfigCache: NewConfigCache(),
		WorkerChan:  make(chan struct{}, config.GetSenderNum()),
	}
}
