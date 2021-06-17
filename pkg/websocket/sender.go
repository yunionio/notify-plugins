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

package websocket

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/notify-plugin/pkg/common"
)

type SWebsocketSender struct {
	common.SSenderBase
	region     string
	clientLock sync.RWMutex // lock to protect client
	session    *mcclient.ClientSession
}

func (self *SWebsocketSender) IsReady(ctx context.Context) bool {
	return self.session != nil
}

func (self *SWebsocketSender) UpdateConfig(ctx context.Context, configs map[string]string) error {
	self.ConfigCache.BatchSet(configs)
	return self.initClient()
}

func (self *SWebsocketSender) FetchContact(ctx context.Context, related string) (string, error) {
	return "", nil
}

func (self *SWebsocketSender) Send(ctx context.Context, params *common.SendParam) error {
	return self.Do(func() error {
		return self.send(params)
	})
}

func (self *SWebsocketSender) BatchSend(ctx context.Context, params *common.BatchSendParam) ([]*common.FailedRecord, error) {
	return common.BatchSend(ctx, params, self.Send)
}

func NewSender(config common.IServiceOptions) common.ISender {
	return &SWebsocketSender{
		SSenderBase: common.NewSSednerBase(config),
		region:      config.GetOthers().(string),
	}
}

func (self *SWebsocketSender) initClient() error {
	vals, ok, noKey := self.ConfigCache.BatchGet(AUTH_URI, ADMIN_USER, ADMIN_PASSWORD, ADMIN_TENANT_NAME)
	if !ok {
		return errors.Wrap(common.ErrConfigMiss, noKey)
	}
	authUri, adminUser, adminPassword, adminTenantName := vals[0], vals[1], vals[2], vals[3]

	a := auth.NewAuthInfo(authUri, "", adminUser, adminPassword, adminTenantName, "")
	auth.Init(a, false, true, "", "")
	self.clientLock.Lock()
	defer self.clientLock.Unlock()
	self.session = auth.GetAdminSession(context.Background(), self.region, "")
	return nil
}

func (self *SWebsocketSender) refreshClient() {
	self.clientLock.Lock()
	defer self.clientLock.Unlock()
	self.session = auth.GetAdminSession(context.Background(), self.region, "")
}

func (self *SWebsocketSender) send(args *common.SendParam) error {
	// component request body
	body := jsonutils.DeepCopy(params).(*jsonutils.JSONDict)
	body.Add(jsonutils.NewString(args.Title), "action")
	body.Add(jsonutils.NewString(fmt.Sprintf("priority=%s; content=%s", args.Priority, args.Message)), "notes")
	body.Add(jsonutils.NewString(args.Contact), "user_id")
	body.Add(jsonutils.NewString(args.Contact), "user")
	if len(args.Contact) == 0 {
		body.Add(jsonutils.JSONTrue, "broadcast")
	}
	if self.isFailed(args.Title, args.Message) {
		body.Add(jsonutils.JSONFalse, "success")
	}
	self.clientLock.RLock()
	session := self.session
	self.clientLock.RUnlock()
	_, err := modules.Websockets.Create(session, body)
	if err != nil {
		// failed
		self.refreshClient()
		self.clientLock.RLock()
		session = self.session
		self.clientLock.RUnlock()
		_, err = modules.Websockets.Create(session, body)

		return err
	}
	return nil
}

func (self *SWebsocketSender) isFailed(title, message string) bool {
	for _, c := range []string{title, message} {
		for _, k := range FAIL_KEY {
			if strings.Contains(c, k) {
				return true
			}
		}
	}
	return false
}
