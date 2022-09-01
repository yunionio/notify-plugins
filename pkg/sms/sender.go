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

package sms

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/notify-plugins/pkg/common"
	"yunion.io/x/notify-plugins/pkg/sms/driver"
)

type SSMSSender struct {
	common.SSenderBase
}

func (self *SSMSSender) IsReady(ctx context.Context) bool {
	return true
}

func (self *SSMSSender) UpdateConfig(ctx context.Context, configs map[string]string) error {
	for key, value := range configs {
		if key == ACESS_KEY_SECRET_BP {
			key = driver.ACCESS_KEY_SECRET
		}
		if key == ACESS_KEY_ID_BP {
			key = driver.ACCESS_KEY_ID
		}
		log.Debugf("update config: %s: %s", key, value)
		self.ConfigCache.Set(key, value)
	}
	drv := driver.GetDriver(self.ConfigCache.Map())
	if drv == nil {
		return driver.ErrDriverNotFound
	}
	err := drv.Verify(ctx, self.ConfigCache.Map())
	return errors.Wrap(err, "Verify")
}

func ValidateConfig(ctx context.Context, configs map[string]string) (bool, string, error) {
	drv := driver.GetDriver(configs)
	if drv == nil {
		return false, "Invalid driver", driver.ErrDriverNotFound
	}
	err := drv.Verify(ctx, configs)
	if err == nil {
		return true, "", nil
	}
	return false, err.Error(), err
}

func (self *SSMSSender) Send(ctx context.Context, params *common.SendParam) error {
	if len(params.RemoteTemplate) == 0 {
		return errors.Wrapf(common.ErrConfigMiss, "require remoteTemplate")
	}
	log.Debugf("reomte template: %s", params.RemoteTemplate)
	err := self.Do(func() error {
		drv := driver.GetDriver(self.ConfigCache.Map())
		if drv == nil {
			return driver.ErrDriverNotFound
		}
		p := make([][]string, 0)
		pJson, err := jsonutils.ParseString(params.Message)
		if err != nil {
			return errors.Wrap(err, "ParseString")
		}
		err = pJson.Unmarshal(&p)
		if err != nil {
			return errors.Wrap(err, "Unmarshal")
		}
		err = drv.Send(ctx, self.ConfigCache.Map(), params.Contact, params.RemoteTemplate, p)
		if err != nil {
			return errors.Wrap(err, "Send")
		}
		return nil
	})
	if err != nil {
		log.Errorf("send fail %s", err)
	}
	return errors.Wrap(err, "Do")
}

func (self *SSMSSender) BatchSend(ctx context.Context, params *common.BatchSendParam) ([]*common.FailedRecord, error) {
	return common.BatchSend(ctx, params, self.Send)
}

func NewSender(config common.IServiceOptions) common.ISender {
	return &SSMSSender{
		SSenderBase: common.NewSSednerBase(config),
	}
}
