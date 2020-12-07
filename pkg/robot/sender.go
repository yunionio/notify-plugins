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
	webhooks := strings.Split(vals[0], ";")
	for i := range webhooks {
		webhooks[i] = strings.TrimSpace(webhooks[i])
	}
	return webhooks, nil
}

func (self *SRebotSender) UpdateConfig(ctx context.Context, configs map[string]string) error {
	config, _ := configs[WEBHOOK]
	urls := strings.Split(config, ";")
	webhooks := make([]string, len(urls))
	for i, url := range urls {
		token := self.tokenFromWebhookUrl(url)
		if len(token) == 0 {
			return fmt.Errorf("invalid webhook: %s", url)
		}
		webhooks[i] = token
	}
	self.ConfigCache.BatchSet(map[string]string{WEBHOOK: strings.Join(webhooks, ";")})
	return nil
}

func (self *SRebotSender) tokenFromWebhookUrl(url string) string {
	for _, prefix := range self.WebhookPrefixs {
		if strings.HasPrefix(url, prefix) {
			return url[len(prefix):]
		}
	}
	return ""
}

func (self *SRebotSender) ValidateConfig(ctx context.Context, configs interface{}) (isValid bool, msg string, err error) {
	webhooks := configs.([]string)
	for _, webhook := range webhooks {
		urlOrToken := self.tokenFromWebhookUrl(webhook)
		if len(urlOrToken) == 0 {
			isValid, msg, err = false, fmt.Sprintf("invalid webhook: %s", webhook), nil
			return
		}
		err = self.send(ctx, urlOrToken, "Validate", "This is a validate message.", []string{})
		if err == ErrNoSuchWebhook {
			isValid = false
			err = nil
			msg = fmt.Sprintf("Invalid webhook %q", webhook)
			break
		}
		if err != nil {
			return isValid, msg, err
		}
		isValid = true
	}
	return isValid, msg, nil
}

func (self *SRebotSender) Send(ctx context.Context, params *apis.SendParams) error {
	config, _ := self.ConfigCache.Get(WEBHOOK)
	webhooks := strings.Split(config, ";")
	// separate contacts
	contacts := strings.Split(params.Contact, ",")
	for i := range contacts {
		contacts[i] = strings.TrimSpace(contacts[i])
	}
	for _, webhook := range webhooks {
		err := self.send(ctx, webhook, params.Title, params.Message, contacts)
		if err != nil {
			return errors.Wrapf(err, "unable to send message to webhook %q", webhook)
		}
	}
	return nil
}

func (self *SRebotSender) BatchSend(ctx context.Context, params *apis.BatchSendParams) ([]*apis.FailedRecord, error) {
	config, _ := self.ConfigCache.Get(WEBHOOK)
	webhooks := strings.Split(config, ";")
	for _, webhook := range webhooks {
		err := self.send(ctx, webhook, params.Title, params.Message, params.Contacts)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to send message to webhook %q", webhook)
		}
	}
	return nil, nil
}
