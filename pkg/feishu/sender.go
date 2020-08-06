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
	"fmt"
	"strings"
	"sync"

	"google.golang.org/grpc/codes"

	"yunion.io/x/onecloud/pkg/monitor/notifydrivers/feishu"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/notify-plugin/pkg/apis"
	"yunion.io/x/notify-plugin/pkg/common"
)

type SConnInfo struct {
	AppID     string
	AppSecret string
}

type SFeishuSender struct {
	common.SSenderBase
	client     *feishu.Tenant
	clientLock sync.Mutex
}

func (self *SFeishuSender) IsReady(ctx context.Context) bool {
	return self.client != nil
}

func (self *SFeishuSender) CheckConfig(ctx context.Context, configs map[string]string) (interface{}, error) {
	vals, ok, noKey := common.CheckMap(configs, APP_ID, APP_SECRET)
	if !ok {
		return nil, fmt.Errorf("require %s", noKey)
	}
	return SConnInfo{vals[0], vals[1]}, nil
}

func (self *SFeishuSender) UpdateConfig(ctx context.Context, configs map[string]string) error {
	self.ConfigCache.BatchSet(configs)
	return self.initClient()
}

func (self *SFeishuSender) ValidateConfig(ctx context.Context, configs interface{}) (isValid bool, msg string, err error) {
	info := configs.(SConnInfo)
	rep, err := feishu.GetTenantAccessTokenInternal(info.AppID, info.AppSecret)
	if err == nil {
		isValid = true
		return
	}
	switch rep.Code {
	case 10003:
		msg = "invalid AppId"
	case 10014:
		msg = "invalid AppSecret"
	}
	return
}

func (self *SFeishuSender) FetchContact(ctx context.Context, related string) (string, error) {
	return self.userIdByMobile(related)
}

func (self *SFeishuSender) Send(ctx context.Context, params *apis.SendParams) error {
	return self.Do(func() error {
		return self.send(params)
	})
}

func (self *SFeishuSender) BatchSend(ctx context.Context, params *apis.BatchSendParams) ([]*apis.FailedRecord, error) {
	self.WorkerChan <- struct{}{}
	defer func() {
		<-self.WorkerChan
	}()
	return self.batchSend(params)
}

func init() {
	common.RegisterErr(errors.ErrNotFound, codes.NotFound)
}

func NewSender(config common.IServiceOptions) common.ISender {
	return &SFeishuSender{
		SSenderBase: common.NewSSednerBase(config),
	}
}

func (self *SFeishuSender) batchSend(args *apis.BatchSendParams) ([]*apis.FailedRecord, error) {
	req := feishu.BatchMsgReq{
		OpenIds: args.Contacts,
		MsgType: "text",
		Content: &feishu.MsgContent{Text: args.Message},
	}
	resp, err := self.client.BatchSendMessage(req)
	if self.needRetry(err) {
		_, err = self.client.BatchSendMessage(req)
	}
	if err != nil {
		return nil, err
	}
	records := make([]*apis.FailedRecord, len(resp.Data.InvalidOpenIds))
	for _, id := range resp.Data.InvalidOpenIds {
		record := &apis.FailedRecord{
			Contact: id,
			Reason:  "invalid userid",
		}
		records = append(records, record)
	}
	return records, nil
}

func (self *SFeishuSender) send(args *apis.SendParams) error {
	req := feishu.MsgReq{
		OpenId:  args.Contact,
		MsgType: "text",
		Content: &feishu.MsgContent{Text: args.Message},
	}
	_, err := self.client.SendMessage(req)
	if self.needRetry(err) {
		_, err = self.client.SendMessage(req)
	}
	if err != nil {
		err = errors.Wrap(err, "SendMessage")
	}
	return err
}

func (self *SFeishuSender) initClient() error {
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

func (self *SFeishuSender) userIdByMobile(mobile string) (string, error) {
	userid, err := self.client.UserIdByMobile(mobile)
	if self.needRetry(err) {
		userid, err = self.client.UserIdByMobile(mobile)
	}
	return userid, err
}

func (self *SFeishuSender) needRetry(err error) (retry bool) {
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "99991663") {
		self.clientLock.Lock()
		defer self.clientLock.Unlock()
		err := self.client.RefreshAccessToken()
		if err != nil {
			return
		}
	}
	return true
}
