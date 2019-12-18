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

package smsaliyun

import (
	"errors"
	"sync"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"

	"yunion.io/x/log"

	"notify-plugin/utils"
)

type sConfigCache map[string]string

func newSConfigCache() sConfigCache {
	return make(map[string]string)
}

type sSenderManager struct {
	workerChan  chan struct{}
	templateDir string
	client      *sdk.Client  // client to example sms
	clientLock  sync.RWMutex // lock to protect client

	configCache   sConfigCache   // config cache
	configLock    sync.RWMutex   // lock to protect config cache
}

func newSSenderManager(config *utils.SBaseOptions) *sSenderManager {
	return &sSenderManager{
		workerChan:  make(chan struct{}, config.SenderNum),

		configCache:   newSConfigCache(),
	}
}

func (self *sSenderManager) initClient() {
	self.configLock.RLock()
	accessKeyID, ok := self.configCache[ACCESS_KEY_ID]
	if !ok {
		self.configLock.RUnlock()
		return
	}
	accessKeySecret, ok := self.configCache[ACCESS_KEY_SECRET]
	if !ok {
		self.configLock.RUnlock()
		return
	}
	self.configLock.RUnlock()
	// lock and update
	self.clientLock.Lock()
	defer self.clientLock.Unlock()
	client, err := sdk.NewClientWithAccessKey("default", accessKeyID, accessKeySecret)
	if err != nil {
		log.Errorf("client connect failed because that %s.", err.Error())
		return
	}
	self.client = client
	log.Printf("Total %d workers.", cap(self.workerChan))
}

func (self *sSenderManager) send(req *requests.CommonRequest) error {
	self.clientLock.RLock()
	client := self.client
	self.clientLock.RUnlock()
	reponse, err := client.ProcessCommonRequest(req)
	if err == nil {
		if reponse.IsSuccess() {
			log.Debugf("Sender successfully")
			return nil
		}
		log.Errorf("Send message failed because that %s.", err.Error())
		//todo There may be detailed processing for different errors.
		return errors.New("send error")
	}
	//todo
	self.initClient()
	// try again
	self.clientLock.RLock()
	client = self.client
	self.clientLock.RUnlock()
	reponse, err = client.ProcessCommonRequest(req)
	if err != nil {
		//todo There may be detailed processing for different errors.
		return err
	}
	return nil
}
