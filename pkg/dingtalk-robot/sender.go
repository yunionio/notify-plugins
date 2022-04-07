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

package dingtalk_rebot

import (
	"context"
	"fmt"
	"strings"

	"github.com/hugozhu/godingtalk"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/notify-plugins/pkg/common"
	"yunion.io/x/notify-plugins/pkg/robot"
)

const (
	WEBHOOK_PREFIX = "https://oapi.dingtalk.com/robot/send?access_token="
)

func NewSender(configs common.IServiceOptions) common.ISender {
	return robot.NewSender(configs, Send)
}

func Send(ctx context.Context, webhook, title, msg string) error {
	var atStr strings.Builder
	var token string
	if strings.HasPrefix(webhook, WEBHOOK_PREFIX) {
		token = webhook[len(WEBHOOK_PREFIX):]
	} else {
		return errors.Wrap(robot.InvalidWebhook, webhook)
	}
	processText := fmt.Sprintf("### %s\n%s%s", title, msg, atStr.String())
	atList := &godingtalk.RobotAtList{}
	client := godingtalk.NewDingTalkClient("", "")
	rep, err := client.SendRobotMarkdownAtMessage(token, title, processText, atList)
	if err == nil {
		return nil
	}
	if rep.ErrCode == 310000 {
		if strings.Contains(rep.ErrMsg, "whitelist") {
			return errors.Wrap(ErrIPWhiteList, rep.ErrMsg)
		}
		return ErrNoSupportSecSetting
	}
	if rep.ErrCode == 300001 && strings.Contains(rep.ErrMsg, "token") {
		return robot.ErrNoSuchWebhook
	}
	return err
}
