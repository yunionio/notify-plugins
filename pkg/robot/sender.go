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
	send           SendFunc
	WebhookPrefixs []string
}

func NewSender(config common.IServiceOptions, send SendFunc, prefixs ...string) common.ISender {
	return &SRebotSender{
		SSenderBase:    common.NewSSednerBase(config),
		send:           send,
		WebhookPrefixs: prefixs,
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
	var (
		token string
	)
	for _, prefix := range self.WebhookPrefixs {
		if index := strings.Index(url, prefix); index >= 0 {
			token = url[index+len(prefix):]
			break
		}
	}
	if len(token) == 0 {
		return fmt.Errorf("invalid webhook: %s", url)
	}
	configs[WEBHOOK] = token
	self.ConfigCache.BatchSet(configs)
	return nil
}

func (self *SRebotSender) ValidateConfig(ctx context.Context, configs interface{}) (isValid bool, msg string, err error) {
	var (
		webhook = configs.(string)
		token   string
	)
	for _, prefix := range self.WebhookPrefixs {
		if strings.HasPrefix(webhook, prefix) {
			isValid = true
			token = webhook[strings.Index(webhook, prefix)+len(prefix):]
			break
		}
	}
	if !isValid {
		msg = "Invalid webhook"
		return
	}
	err = self.send(ctx, token, "Validate", "This is a validate message.", []string{})
	if err == ErrNoSuchWebhook {
		msg = "Invalid access token in webhook"
		return
	}
	if err == nil {
		isValid = true
	}
	return
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
