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

package workwx_robot

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/notify-plugin/pkg/common"
	"yunion.io/x/notify-plugin/pkg/robot"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

func NewSender(configs common.IServiceOptions) common.ISender {
	return robot.NewSender(configs, Send, webhookPrefix)
}

func Send(ctx context.Context, token, title, msg string, contacts []string) error {
	req := WebhookTextMsgReq{
		Content:             fmt.Sprintf("%s\n\n%s", title, msg),
		MentionedMobileList: contacts,
	}
	resp, err := sendWebhookTextMessage(context.Background(), token, req)
	if err != nil {
		return err
	}
	if resp.Code == 0 {
		return nil
	}
	if resp.Code == 93000 {
		return robot.ErrNoSuchWebhook
	}
	return fmt.Errorf(resp.Msg)
}

var (
	webhookPrefix = "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key="
	cli           = &http.Client{
		Transport: httputils.GetTransport(true),
	}
)

type WebhookTextMsgReq struct {
	Content             string   `json:"content"`
	MentionedMobileList []string `json:"mentioned_mobile_list"`
}

type WebhookMsgResp struct {
	Code int    `json:"errcode"`
	Msg  string `json:"errmsg"`
}

func sendWebhookTextMessage(ctx context.Context, token string, req WebhookTextMsgReq) (WebhookMsgResp, error) {
	webhook := webhookPrefix + token
	body := jsonutils.NewDict()
	body.Set("msgtype", jsonutils.NewString("text"))
	body.Set("text", jsonutils.Marshal(req))
	_, obj, err := httputils.JSONRequest(cli, ctx, httputils.POST, webhook, http.Header{}, body, false)
	if err != nil {
		return WebhookMsgResp{}, err
	}

	resp := WebhookMsgResp{}
	err = obj.Unmarshal(&resp)
	if err != nil {
		return WebhookMsgResp{}, err
	}
	return resp, nil
}
