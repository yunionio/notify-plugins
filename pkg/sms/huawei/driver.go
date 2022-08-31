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

package huawei

import (
	"context"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/notify-plugins/pkg/sms/driver"
)

func init() {
	driver.Register(&SHuaweiDriver{})
}

type SHuaweiDriver struct{}

func (d *SHuaweiDriver) Name() string {
	return driver.DriverHuawei
}

func (d *SHuaweiDriver) Verify(ctx context.Context, conf map[string]string) error {
	err := d.Send(ctx, conf, "12345678901", "0123456789/0123456789", nil)
	if err == driver.ErrSignnameInvalid || err == driver.ErrSignatureDoesNotMatch || err == driver.ErrAccessKeyIdNotFound {
		return nil
	}
	return errors.Wrap(err, "Verify")
}

func (d *SHuaweiDriver) Send(ctx context.Context, conf map[string]string, dest string, templateId string, params [][]string) error {
	tmp2 := strings.Split(templateId, "/")
	channel := tmp2[0]
	template := tmp2[1]

	appKey := conf[driver.ACCESS_KEY_ID]
	appSecret := conf[driver.ACCESS_KEY_SECRET]
	signature := conf[driver.SIGNATURE]
	serviceUrl := conf[driver.SERVICE_URL]

	p := make([]string, 0)
	for _, param := range params {
		p = append(p, param[1])
	}
	return sendSms(ctx, appKey, appSecret, serviceUrl, channel, dest, template, p, signature)
}
