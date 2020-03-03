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
	"encoding/json"
	"sync"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	sdkerrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/notify-plugin/common"
)

type sSenderManager struct {
	workerChan  chan struct{}
	client      *sdk.Client  // client to example sms
	clientLock  sync.RWMutex // lock to protect client

	configCache   *common.SConfigCache // config cache
}

func newSSenderManager(config *common.SBaseOptions) *sSenderManager {
	return &sSenderManager{
		workerChan:  make(chan struct{}, config.SenderNum),

		configCache: common.NewConfigCache(),
	}
}

func (self *sSenderManager) initClient() error {
	vals, ok, noKey := self.configCache.BatchGet(ACCESS_KEY_ID, ACCESS_KEY_SECRET)
	if !ok {
		return errors.Wrap(common.ErrConfigMiss, noKey)
	}
	accessKeyID, accessKeySecret := vals[0], vals[1]

	// lock and update
	client, err := sdk.NewClientWithAccessKey("default", accessKeyID, accessKeySecret)
	if err != nil {
		return err
	}
	self.clientLock.Lock()
	defer self.clientLock.Unlock()
	self.client = client
	log.Printf("Total %d workers.", cap(self.workerChan))
	return nil
}

func (self *sSenderManager) send(client *sdk.Client, signature, templateCode, templateParam, phoneNumber string, retry bool) error {
	request := requests.NewCommonRequest()
	request.Method = "POST"
	request.Scheme = "https" // https | http
	request.Domain = "dysmsapi.aliyuncs.com"
	request.Version = "2017-05-25"
	request.ApiName = "SendSms"
	request.QueryParams["RegionId"] = "default"
	request.QueryParams["PhoneNumbers"] = phoneNumber
	request.QueryParams["SignName"] = signature

	request.QueryParams["TemplateCode"] = templateCode
	request.QueryParams["TemplateParam"] = templateParam

	if client == nil {
		self.clientLock.RLock()
		client = self.client
		self.clientLock.RUnlock()
	}
	err := self.checkResponseAndError(client.ProcessCommonRequest(request))
	if !retry || err == nil  {
		return err
	}

	self.initClient()
	// try again
	self.clientLock.RLock()
	client = self.client
	self.clientLock.RUnlock()
	return self.checkResponseAndError(client.ProcessCommonRequest(request))
}

func (self *sSenderManager) checkResponseAndError(rep *responses.CommonResponse, err error) error {
	if err != nil {
		serr, ok := err.(*sdkerrors.ServerError)
		if !ok {
			return err
		}
		if serr.ErrorCode() == ACCESSKEYID_NOTFOUND {
			return ErrAccessKeyIdNotFound
		}
		if serr.ErrorCode() == SIGN_DOESNOTMATCH {
			return ErrSignatureDoesNotMatch
		}
		return err
	}

	type RepContent struct {
		Message string
		Code string
	}
	rc := RepContent{}
	err = json.Unmarshal(rep.GetHttpContentBytes(), &rc)
	if err != nil {
		log.Errorf("The Response Content's style may changed")
		return errors.Wrap(err, "json.Unmarshal")
	}
	if rc.Code == "OK" {
		return nil
	}
	if rc.Code == SIGHNTURE_ILLEGAL {
		return ErrSignnameInvalid
	}
	return errors.Error(rc.Message)
}

