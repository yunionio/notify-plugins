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

package robot

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/notify-plugin/pkg/apis"
	"yunion.io/x/notify-plugin/pkg/common"
)

const WEBHOOK = "webhook"

var ErrNoSuchWebhook = errors.Error("No such webhook")

type SendFunc func(ctx context.Context, token, title, msg string, contacts []string) error

type SRebotSender struct {
	common.SSenderBase
	send       SendFunc
	WebhookPrefix string
}

func NewSender(config common.IServiceOptions, send SendFunc, prefix string) common.ISender {
	return &SRebotSender{
		SSenderBase:   common.NewSSednerBase(config),
		send: send,
		WebhookPrefix: prefix,
	}
}

func (self *SRebotSender) IsReady(ctx context.Context) bool {
	_, ok := self.ConfigCache.Get(WEBHOOK)
	return ok
}

func (self *SRebotSender) CheckConfig(ctx context.Context, configs map[string]string) (interface{}, error) {
	vals, ok, nokey := common.CheckMap(configs, WEBHOOK)
	if !ok {
		return nil, fmt.Errorf("require %s", nokey)
	}
	return vals[0], nil
}

func (self *SRebotSender) UpdateConfig(ctx context.Context, configs map[string]string) error {
	url, _ := configs[WEBHOOK]
	if index := strings.Index(url, self.WebhookPrefix); index >= 0 {
		configs[WEBHOOK] = url[index+len(self.WebhookPrefix):]
	}
	self.ConfigCache.BatchSet(configs)
	return nil
}

func (self *SRebotSender) ValidateConfig(ctx context.Context, configs interface{}) (*apis.ValidateConfigReply, error) {
	rep := &apis.ValidateConfigReply{IsValid: false}
	webhook := configs.(string)
	if !strings.HasPrefix(webhook, self.WebhookPrefix) {
		rep.Msg = "Invalid webhook"
	}
	token := webhook[strings.Index(webhook, self.WebhookPrefix)+len(self.WebhookPrefix):]
	err := self.send(ctx, token, "Validate", "This is a validate message.", []string{})
	if err == ErrNoSuchWebhook {
		rep.Msg = "Invalid access token in webhook"
	}
	if err != nil {
		return nil, err
	}
	rep.IsValid = true
	return rep, nil
}

func (self *SRebotSender) Send(ctx context.Context, params *apis.SendParams) error {
	webhook, _ := self.ConfigCache.Get(WEBHOOK)
	// separate contacts
	contacts := strings.Split(params.Contact, ",")
	for i := range contacts {
		contacts[i] = strings.TrimSpace(contacts[i])
	}
	return self.send(ctx, webhook, params.Title, params.Message, contacts)
}
