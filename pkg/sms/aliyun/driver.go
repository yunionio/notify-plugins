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

package aliyun

import (
	"context"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/notify-plugins/pkg/sms/driver"
)

func init() {
	driver.Register(&SAliyunDriver{})
}

type SAliyunDriver struct{}

func (d *SAliyunDriver) Name() string {
	return driver.DriverAliyun
}

func (d *SAliyunDriver) Verify(ctx context.Context, conf map[string]string) error {
	err := d.Send(ctx, conf, "12345678901", "SMS_123456789", nil)
	if err == driver.ErrSignnameInvalid || err == driver.ErrSignatureDoesNotMatch || err == driver.ErrAccessKeyIdNotFound {
		return nil
	}
	return errors.Wrap(err, "Verify")
}

func (d *SAliyunDriver) Send(ctx context.Context, conf map[string]string, dest string, templateId string, params [][]string) error {
	appKey := conf[driver.ACCESS_KEY_ID]
	appSecret := conf[driver.ACCESS_KEY_SECRET]
	signature := conf[driver.SIGNATURE]

	p := make(map[string]string)
	for _, param := range params {
		p[param[0]] = param[1]
	}
	return sendSms(appKey, appSecret, signature, templateId, p, dest)
}
