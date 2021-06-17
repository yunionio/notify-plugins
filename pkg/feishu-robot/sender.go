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

package feishu_robot

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/onecloud/pkg/monitor/notifydrivers/feishu"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/notify-plugin/pkg/common"
	"yunion.io/x/notify-plugin/pkg/robot"
)

const (
	ApiWebhookRobotV2SendMessage = "https://open.feishu.cn/open-apis/bot/v2/hook/"
)

func NewSender(configs common.IServiceOptions) common.ISender {
	return robot.NewSender(configs, Send)
}

func Send(ctx context.Context, webhook, title, msg string) error {
	var token string
	switch {
	case strings.HasPrefix(webhook, ApiWebhookRobotV2SendMessage):
		token = webhook[len(ApiWebhookRobotV2SendMessage):]
	case strings.HasPrefix(webhook, feishu.ApiWebhookRobotSendMessage):
		token = webhook[len(feishu.ApiWebhookRobotSendMessage):]
	default:
		return errors.Wrap(robot.InvalidWebhook, webhook)
	}
	req := feishu.WebhookRobotMsgReq{
		Title: title,
		Text:  msg,
	}
	rep, err := feishu.SendWebhookRobotMessage(token, req)
	if err != nil {
		return errors.Wrap(err, "SendWebhookRobotMessage")
	}
	if !rep.Ok {
		if strings.Contains(rep.Error, "token") {
			return robot.ErrNoSuchWebhook
		} else {
			return fmt.Errorf("SendWebhookRobotMessage failed: %s", rep.Error)
		}
	}
	return err
}
