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

package webhook

import (
	"context"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/util/httputils"

	"yunion.io/x/notify-plugin/pkg/common"
	"yunion.io/x/notify-plugin/pkg/robot"
)

func NewSender(configs common.IServiceOptions) common.ISender {
	option := configs.(*SOptions)
	initHttpClient(option.Insecure)
	return robot.NewSender(configs, Send, "")
}

var (
	cli *http.Client
)

func initHttpClient(insecure bool) {
	cli = &http.Client{
		Transport: httputils.GetTransport(false),
	}
}

const (
	EVENT_HEADER = "X-Yunion-Event"
)

func Send(ctx context.Context, webhook, event, msg string, contacts []string) error {
	log.Infof("event: %s, msg: %s", event, msg)
	body, err := jsonutils.ParseString(msg)
	if err != nil {
		log.Errorf("unable to parse %q: %v", msg, err)
	}
	if _, ok := body.(*jsonutils.JSONString); err != nil || ok {
		dict := jsonutils.NewDict()
		dict.Set("Msg", jsonutils.NewString(msg))
		body = dict
	}
	event = strings.ToUpper(event)
	header := http.Header{}
	header.Set(EVENT_HEADER, event)
	_, _, err = httputils.JSONRequest(cli, ctx, httputils.POST, webhook, header, body, false)
	return err
}
