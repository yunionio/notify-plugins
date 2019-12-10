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
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"

	"notify-plugin/pkg/apis"
)

var senderManager *sSenderManager

type sConfigCache map[string]string

func newSConfigCache() sConfigCache {
	return make(map[string]string)
}

type sSenderManager struct {
	workerChan  chan struct{}
	templateDir string
	region      string
	clientLock  sync.RWMutex // lock to protect client
	session     *mcclient.ClientSession

	configCache   sConfigCache   // config cache
	configLock    sync.RWMutex   // lock to protect config cache
}

func newSSenderManager(config *SWebsocketConfig) *sSenderManager {
	return &sSenderManager{
		workerChan:  make(chan struct{}, config.SenderNum),
		region:      config.Region,

		configCache:   newSConfigCache(),
	}
}

func (self *sSenderManager) initClient() {
	self.configLock.RLock()
	authUri, ok := self.configCache[AUTH_URI]
	if !ok {
		self.configLock.RUnlock()
		return
	}
	adminUser, ok := self.configCache[ADMIN_USER]
	if !ok {
		self.configLock.RUnlock()
		return
	}
	adminPassword, ok := self.configCache[ADMIN_PASSWORD]
	if !ok {
		self.configLock.RUnlock()
		return
	}
	adminTenantName, ok := self.configCache[ADMIN_TENANT_NAME]
	self.configLock.RUnlock()

	self.clientLock.Lock()
	defer self.clientLock.Unlock()
	a := auth.NewAuthInfo(authUri, "", adminUser, adminPassword, adminTenantName, "")
	auth.Init(a, false, true, "", "")
	self.session = auth.GetAdminSession(context.Background(), self.region, "")
}

func (self *sSenderManager) send(args *apis.SendParams) error {
	// component request body
	body := jsonutils.DeepCopy(params).(*jsonutils.JSONDict)
	body.Add(jsonutils.NewString(args.Title), "action")
	body.Add(jsonutils.NewString(fmt.Sprintf("priority=%s; content=%s", args.Priority, args.Message)), "notes")
	body.Add(jsonutils.NewString(args.Contact), "user_id")
	body.Add(jsonutils.NewString(args.Contact), "user")
	if len(args.Contact) == 0 {
		body.Add(jsonutils.JSONTrue, "broadcast")
	}
	self.clientLock.RLock()
	session := self.session
	self.clientLock.RUnlock()
	_, err := modules.Websockets.Create(session, body)
	if err != nil {
		// failed
		self.initClient()
		self.clientLock.RLock()
		session = self.session
		self.clientLock.RUnlock()
		_, err = modules.Websockets.Create(session, body)

		return err
	}
	return nil
}
