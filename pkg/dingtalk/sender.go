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
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/hugozhu/godingtalk"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/notify-plugin/pkg/apis"
	"yunion.io/x/notify-plugin/pkg/common"
)

type SConnInfo struct {
	AgentID   string
	AppKey    string
	AppSecret string
}

type sSendFunc func(*godingtalk.DingTalkClient, string) error

type SDingtalkSender struct {
	common.SSenderBase
	client     *godingtalk.DingTalkClient // client to example sms
	clientLock sync.Mutex                 // lock to protect client
}

func (self *SDingtalkSender) IsReady(ctx context.Context) bool {
	return self.client == nil
}

func (self *SDingtalkSender) CheckConfig(ctx context.Context, configs map[string]string) (interface{}, error) {
	vals, ok, noKey := common.CheckMap(configs, AGENT_ID, APP_KEY, APP_SECRET)
	if !ok {
		return nil, fmt.Errorf("require %s", noKey)
	}
	return SConnInfo{vals[0], vals[1], vals[2]}, nil
}

func (self *SDingtalkSender) UpdateConfig(ctx context.Context, configs map[string]string) error {
	self.ConfigCache.BatchSet(configs)
	return self.initClient()
}

func (self *SDingtalkSender) ValidateConfig(ctx context.Context, configs interface{}) (*apis.ValidateConfigReply, error) {
	info := configs.(SConnInfo)
	cache_file := fmt.Sprintf(".%s_validate", info.AppKey)
	defer os.Remove(cache_file)
	client := godingtalk.NewDingTalkClient(info.AppKey, info.AppSecret)

	//hack
	client.Cache = godingtalk.NewFileCache(cache_file)
	ret := &apis.ValidateConfigReply{}
	err := client.RefreshAccessToken()
	if err != nil {
		if strings.Contains(err.Error(), "40089") {
			ret.Msg = "invalid AppKey or AppSecret"
			return ret, nil
		}
		return nil, err
	}
	ret.IsValid = true
	return ret, nil
}

func (self *SDingtalkSender) FetchContact(ctx context.Context, related string) (string, error) {
	return self.getUseridByMobile(related)
}

func (self *SDingtalkSender) Send(ctx context.Context, params *apis.SendParams) error {
	sendFunc := self.getSendFunc(params)
	return self.Do(func() error {
		return self.send(sendFunc)
	})
}

func NewSender(config common.IServiceOptions) common.ISender {
	return &SDingtalkSender{
		SSenderBase: common.NewSSednerBase(config),
	}
}

func (self *SDingtalkSender) getSendFunc(args *apis.SendParams) sSendFunc {
	if args.Title == args.Topic {
		return func(client *godingtalk.DingTalkClient, agentID string) error {
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
	return func(client *godingtalk.DingTalkClient, agentID string) error {
		err := client.SendAppOAMessage(agentID, args.Contact, message)
		if err != nil {
			return fmt.Errorf("UserIDs: %s: %w", args.Contact, err)
		}
		return nil
	}
}

func (self *SDingtalkSender) getUseridByMobile(mobile string) (string, error) {
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

func (self *SDingtalkSender) initClient() error {
	vals, ok, noKey := self.ConfigCache.BatchGet(APP_KEY, APP_SECRET)
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

func (self *SDingtalkSender) send(sendFunc sSendFunc) error {
	// get agentID
	agentID, ok := self.ConfigCache.Get(AGENT_ID)
	if !ok {
		return ErrAgentIDNotInit
	}

	// example message
	err := sendFunc(self.client, agentID)
	if err == nil {
		log.Debugf("send message successfully.")
		return nil
	}
	return errors.Wrap(err, "send failed")
}
