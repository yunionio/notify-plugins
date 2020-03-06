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

package dingtalk

import (
	"fmt"
	"sync"
	"yunion.io/x/pkg/errors"

	"github.com/hugozhu/godingtalk"

	"yunion.io/x/log"

	"yunion.io/x/notify-plugin/pkg/apis"
	"yunion.io/x/notify-plugin/common"
)

var senderManager *sSenderManager

type sSendFunc func(*sSenderManager, string) error

type sSenderManager struct {
	workerChan  chan struct{}
	client      *godingtalk.DingTalkClient // client to example sms
	clientLock  sync.RWMutex               // lock to protect client

	configCache *common.SConfigCache // config cache
}

func newSSenderManager(config *common.SBaseOptions) *sSenderManager {
	return &sSenderManager{
		workerChan:  make(chan struct{}, config.SenderNum),
		configCache: common.NewConfigCache(),
	}
}

func (self *sSenderManager) getSendFunc(args *apis.SendParams) sSendFunc {
	if args.Title == args.Topic {
		return func(manager *sSenderManager, agentID string) error {
			manager.clientLock.RLock()
			client := manager.client
			manager.clientLock.RUnlock()
			err := client.SendAppMessage(agentID, args.Contact, args.Message)
			if err != nil {
				return fmt.Errorf("UserIDs: %s: %w", args.Contact, err)
			}
			return nil
		}
	}
	message := godingtalk.OAMessage{}
	message.Head.Text = args.Topic
	message.Body.Title = args.Title
	message.Body.Content = args.Message
	return func(manager *sSenderManager, agentID string) error {
		manager.clientLock.RLock()
		client := manager.client
		manager.clientLock.RUnlock()
		err := client.SendAppOAMessage(agentID, args.Contact, message)
		if err != nil {
			return fmt.Errorf("UserIDs: %s: %w", args.Contact, err)
		}
		return nil
	}
}

func (self *sSenderManager) getUseridByMobile(mobile string) (string, error) {
	// get department list
	userid, err := self.client.UseridByMobile(mobile)
	if err != nil {
		return "", err
	}
	if len(userid) == 0 {
		return "", ErrNoSuchUser
	}
	return userid, nil
}

func (self *sSenderManager) initClient() error {
	vals, ok, noKey := self.configCache.BatchGet(APP_KEY, APP_SECRET)
	if !ok {
		return errors.Wrap(common.ErrConfigMiss, noKey)
	}
	appKey, appSecret := vals[0], vals[1]

	// lock and update
	client := godingtalk.NewDingTalkClient(appKey, appSecret)
	err := client.RefreshAccessToken()
	if err != nil {
		return err
	}
	self.clientLock.Lock()
	defer self.clientLock.Unlock()
	self.client = client
	return nil
}

func (self *sSenderManager) send(sendFunc sSendFunc) error {
	// get agentID
	agentID, ok := self.configCache.Get(AGENT_ID)
	if !ok {
		return ErrAgentIDNotInit
	}

	// example message
	err := sendFunc(self, agentID)
	if err == nil {
		log.Debugf("send message successfully.")
		return nil
	}

	// access_token must not be expired
	//if strings.Contains(err.Error(), "access_token") || strings.Contains(err.Error(), "accessToken") {
	//	self.initClient()
	//	// try again
	//	err = sendFunc(self, agentID)
	//	if err != nil {
	//		fmt.Errorf("send failed after fetch access_token again: %w", err)
	//	}
	//	log.Debugf("send message successfully.")
	//	return nil
	//}
	return errors.Wrap(err, "send failed")
}
