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

package feishu

import (
	"context"
	"sync"

	"google.golang.org/grpc/codes"

	"yunion.io/x/onecloud/pkg/monitor/notifydrivers/feishu"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/notify-plugin/pkg/apis"
	"yunion.io/x/notify-plugin/pkg/common"
)

type SSendManager struct {
	common.SSenderBase
	client     *feishu.Tenant
	clientLock sync.RWMutex
}

func (self *SSendManager) IsReady(ctx context.Context) bool {
	return self.client == nil
}

func (self *SSendManager) CheckConfig(ctx context.Context, configs map[string]string) (interface{}, error) {
	return nil, nil
}

func (self *SSendManager) UpdateConfig(ctx context.Context, configs map[string]string) error {
	self.ConfigCache.BatchSet(configs)
	return self.initClient()
}

func (self *SSendManager) ValidateConfig(ctx context.Context, configs interface{}) (*apis.ValidateConfigReply, error) {
	return nil, nil
}

func (self *SSendManager) FetchContact(ctx context.Context, related string) (string, error) {
	return self.userIdByMobile(related)
}

func (self *SSendManager) Send(ctx context.Context, params *apis.SendParams) error {
	return self.Do(func() error {
		return self.send(params)
	})
}

func init() {
	common.RegisterErr(errors.ErrNotFound, codes.NotFound)
}

func NewSender(config common.IServiceOptions) common.ISender {
	return &SSendManager{
		SSenderBase: common.NewSSednerBase(config),
	}
}

func (self *SSendManager) send(args *apis.SendParams) error {
	req := feishu.MsgReq{
		OpenId:  args.Contact,
		MsgType: "text",
		Content: &feishu.MsgContent{Text: args.Message},
	}
	_, err := self.client.SendMessage(req)
	if err != nil {
		err = errors.Wrap(err, "SendMessage")
	}
	return err
}

func (self *SSendManager) initClient() error {
	vals, ok, noKey := self.ConfigCache.BatchGet(APP_ID, APP_SECRET)
	if !ok {
		return errors.Wrap(common.ErrConfigMiss, noKey)
	}
	appID, appSecret := vals[0], vals[1]

	// lock and update
	client, err := feishu.NewTenant(appID, appSecret)
	if err != nil {
		return err
	}
	self.clientLock.Lock()
	defer self.clientLock.Unlock()
	self.client = client
	return nil
}

func (self *SSendManager) userIdByMobile(mobile string) (string, error) {
	return self.client.UserIdByMobile(mobile)
}
