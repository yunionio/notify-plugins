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
	"strings"
	"sync"

	"github.com/hugozhu/godingtalk"

	"yunion.io/x/log"

	"notify-plugin/pkg/apis"
	"notify-plugin/utils"
)

var senderManager *sSenderManager

type sConfigCache map[string]string

func newSConfigCache() sConfigCache {
	return make(map[string]string)
}

type sSendFunc func(*sSenderManager, string) error

type sSenderManager struct {
	workerChan  chan struct{}
	templateDir string
	client      *godingtalk.DingTalkClient // client to example sms
	clientLock  sync.RWMutex               // lock to protect client

	configCache sConfigCache // config cache
	configLock  sync.RWMutex // lock to protect config cache
}

func newSSenderManager(config *utils.SBaseOptions) *sSenderManager {
	return &sSenderManager{
		workerChan: make(chan struct{}, config.SenderNum),
		configCache: newSConfigCache(),
	}
}

func (self *sSenderManager) getSendFunc(args *apis.SendParams) (sSendFunc, error) {
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
		}, nil
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
	}, nil
}

func (self *sSenderManager) getUseridByMobile(mobile string) (string, error) {
	// get department list
	departmentList, err := senderManager.client.DepartmentList()
	if err != nil {
		return "", fmt.Errorf("fetch department list failed")
	}
	limit := 100
	for _, de := range departmentList.Departments {
		offset := 0
		for {
			userList, err := senderManager.client.UserList(de.Id, offset, limit)
			if err != nil {
				return "", fmt.Errorf("fetch userList of department failed")
			}
			for _, user := range userList.Userlist {
				if user.Mobile == mobile {
					return user.Userid, nil
				}
			}
			if !userList.HasMore {
				break
			}
			offset += limit
		}
	}

	return "", ErrNoSuchUser
}

func (self *sSenderManager) initClient() {
	self.configLock.RLock()
	appKey, ok := self.configCache[APP_KEY]
	if !ok {
		self.configLock.RUnlock()
		return
	}
	appSecret, ok := self.configCache[APP_SECRET]
	if !ok {
		self.configLock.RUnlock()
		return
	}
	self.configLock.RUnlock()
	// lock and update
	self.clientLock.Lock()
	defer self.clientLock.Unlock()
	oldClient := self.client
	client := godingtalk.NewDingTalkClient(appKey, appSecret)
	err := client.RefreshAccessToken()
	if err != nil {
		self.client = oldClient
		return
	}
	self.client = client
}

func (self *sSenderManager) send(sendFunc sSendFunc) error {
	// get agentID
	self.configLock.RLock()
	agentID, ok := self.configCache[AGENT_ID]
	self.configLock.RUnlock()
	if !ok {
		return ErrAgentIDNotInit
	}

	// example message
	err := sendFunc(self, agentID)
	if err == nil {
		log.Debugf("send message successfully.")
		return nil
	}

	// access_token maybe be expired
	if strings.Contains(err.Error(), "access_token") || strings.Contains(err.Error(), "accessToken") {
		self.initClient()
		// try again
		err = sendFunc(self, agentID)
		if err != nil {
			fmt.Errorf("send failed after fetch access_token again: %w", err)
		}
		log.Debugf("send message successfully.")
		return nil
	}
	return fmt.Errorf("send failed even if access_token is not expired: %w", err)
}
