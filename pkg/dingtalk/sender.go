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
	"time"

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
	return self.client != nil
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

func (self *SDingtalkSender) ValidateConfig(ctx context.Context, configs interface{}) (isValid bool, msg string, err error) {
	info := configs.(SConnInfo)
	cache_file := fmt.Sprintf(".%s_validate", info.AppKey)
	defer os.Remove(cache_file)
	client := godingtalk.NewDingTalkClient(info.AppKey, info.AppSecret)

	//hack
	client.Cache = godingtalk.NewFileCache(cache_file)
	err = client.RefreshAccessToken()
	if err != nil {
		if strings.Contains(err.Error(), "40089") {
			msg, err = "invalid AppKey or AppSecret", nil
			return
		}
		return
	}
	isValid = true
	return
}

func (self *SDingtalkSender) FetchContact(ctx context.Context, related string) (string, error) {
	return self.getUseridByMobile(related)
}

func (self *SDingtalkSender) Send(ctx context.Context, params *apis.SendParams) error {
	sendFunc := func() error {
		batchParams := &apis.BatchSendParams{
			Contacts:       []string{params.Contact},
			Title:          params.Title,
			Message:        params.Message,
			Priority:       params.Priority,
			RemoteTemplate: params.RemoteTemplate,
		}
		_, err := self.batchSendTopMsg(batchParams)
		return err
	}
	return self.Do(sendFunc)
}

func (self *SDingtalkSender) BatchSend(ctx context.Context, params *apis.BatchSendParams) ([]*apis.FailedRecord, error) {
	self.WorkerChan <- struct{}{}
	defer func() {
		<-self.WorkerChan
	}()
	return self.batchSendTopMsg(params)
}

func NewSender(config common.IServiceOptions) common.ISender {
	return &SDingtalkSender{
		SSenderBase: common.NewSSednerBase(config),
	}
}

func (self *SDingtalkSender) batchSendTopMsg(args *apis.BatchSendParams) ([]*apis.FailedRecord, error) {
	initInterval := time.Duration(len(args.Contacts)/5) * time.Second
	getInterval := func() time.Duration {
		return initInterval + time.Second
	}
	msg := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"title": args.GetTitle(),
			"text":  args.GetMessage(),
		},
	}
	taskID, err := self.client.TopAPIMsgSendv2(args.GetContacts(), msg)
	if err != nil {
		return nil, err
	}
	for {
		time.Sleep(getInterval())
		progress, err := self.client.TopAPIMsgGetSendProgressv2(taskID)
		if err != nil {
			log.Errorf("fail to get progress for task(%d): %v", taskID, err)
		}
		if progress.Status == 2 {
			break
		}
		if progress.Status == 0 {
			log.Debugf("Task(%d), progress: 0%%", taskID)
			continue
		}
		if progress.Status == 1 {
			log.Debugf("Task(%d), progress: %d%%", taskID, progress.Percent)
		}
	}
	result, err := self.client.TopAPIMsgGetSendResultv2(taskID)
	if err != nil {
		return nil, err
	}
	ret := make([]*apis.FailedRecord, 0, len(result.FaildedUserIDList)+len(result.ForbiddenUserIDList)+len(result.InvalidUserIDList))
	for _, id := range result.FaildedUserIDList {
		record := &apis.FailedRecord{
			Contact: id,
			Reason:  "",
		}
		ret = append(ret, record)
	}
	for _, id := range result.ForbiddenUserIDList {
		record := &apis.FailedRecord{
			Contact: id,
			Reason:  "forbidden user",
		}
		ret = append(ret, record)
	}
	for _, id := range result.InvalidUserIDList {
		record := &apis.FailedRecord{
			Contact: id,
			Reason:  "invalid userid",
		}
		ret = append(ret, record)
	}
	return ret, nil
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
	if self.needRetry(err) {
		userid, err = self.client.UseridByMobile(mobile)
	}
	if err == nil {
		return userid, nil
	}
	oErr, ok := err.(godingtalk.OAPIError)
	if !ok {
		return "", err
	}
	if oErr.ErrCode == 60121 {
		return "", errors.Wrap(common.ErrNoSuchMobile, err.Error())
	}
	if oErr.ErrCode == 60020 || oErr.ErrCode == 60011 {
		return "", errors.Wrap(common.ErrIncompleteConfig, err.Error())
	}
	return "", err
}

func (self *SDingtalkSender) initClient() error {
	vals, ok, noKey := self.ConfigCache.BatchGet(APP_KEY, APP_SECRET, AGENT_ID)
	if !ok {
		return errors.Wrap(common.ErrConfigMiss, noKey)
	}
	appKey, appSecret, agentID := vals[0], vals[1], vals[2]

	// lock and update
	client := godingtalk.NewDingTalkClient(appKey, appSecret).WithAgentID(agentID)
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
	if self.needRetry(err) {
		err = sendFunc(self.client, agentID)
	}
	if err == nil {
		return nil
	}
	return errors.Wrap(err, "send failed")
}

func (self *SDingtalkSender) needRetry(err error) (retry bool) {
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "access_token") {
		self.clientLock.Lock()
		defer self.clientLock.Unlock()
		err := self.client.RefreshAccessToken()
		if err != nil {
			return
		}
	}
	return true
}
