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
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sync"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	sdkerrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/notify-plugins/pkg/common"
)

type SConnectInfo struct {
	AccessKeyID     string
	AccessKeySecret string
	Signature       string
}

type SSMSAliyunSender struct {
	common.SSenderBase
	client     *sdk.Client  // client to example sms
	clientLock sync.RWMutex // lock to protect client
}

func (self *SSMSAliyunSender) IsReady(ctx context.Context) bool {
	return self.client != nil
}

func (self *SSMSAliyunSender) UpdateConfig(ctx context.Context, configs map[string]string) error {
	for key, value := range configs {
		if key == ACESS_KEY_SECRET_BP {
			key = ACCESS_KEY_SECRET
		}
		if key == ACESS_KEY_ID_BP {
			key = ACCESS_KEY_ID
		}
		log.Debugf("update config: %s: %s", key, value)
		self.ConfigCache.Set(key, value)
	}
	return self.initClient()
}

func ValidateConfig(ctx context.Context, configs map[string]string) (isValid bool, msg string, err error) {
	vals, ok, noKey := common.CheckMap(configs, ACCESS_KEY_ID, ACCESS_KEY_SECRET, SIGNATURE)
	if !ok {
		err = fmt.Errorf("require %s", noKey)
		return
	}
	id, secret, signature := vals[0], vals[1], vals[2]
	client, err := sdk.NewClientWithAccessKey("default", id, secret)
	if err != nil {
		err = errors.Wrap(err, "NewClientWithAccessKey")
		return
	}
	err = send(client, signature, "SMS_123456789", `{"code": "123456"}`, "12345678901")
	if err == ErrSignnameInvalid || err == ErrSignatureDoesNotMatch || err == ErrAccessKeyIdNotFound {
		msg, err = err.Error(), nil
		return
	}
	isValid, err = true, nil
	return
}

func (self *SSMSAliyunSender) Send(ctx context.Context, params *common.SendParam) error {
	signature, _ := self.ConfigCache.Get(SIGNATURE)
	if len(params.RemoteTemplate) == 0 {
		return errors.Wrapf(common.ErrConfigMiss, "require remoteTemplate")
	}
	log.Debugf("reomte template: %s", params.RemoteTemplate)
	return self.Do(func() error {
		return self.send(signature, params.RemoteTemplate, params.Message, params.Contact, true)
	})
}

func (self *SSMSAliyunSender) BatchSend(ctx context.Context, params *common.BatchSendParam) ([]*common.FailedRecord, error) {
	return common.BatchSend(ctx, params, self.Send)
}

func NewSender(config common.IServiceOptions) common.ISender {
	return &SSMSAliyunSender{
		SSenderBase: common.NewSSednerBase(config),
	}
}

func (self *SSMSAliyunSender) initClient() error {
	vals, ok, noKey := self.ConfigCache.BatchGet(ACCESS_KEY_ID, ACCESS_KEY_SECRET)
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
	return nil
}

var parser = regexp.MustCompile(`\+(\d*) (.*)`)

func (self *SSMSAliyunSender) send(signature, templateCode, templateParam, phoneNumber string, retry bool) error {
	self.clientLock.RLock()
    client := self.client
	self.clientLock.RUnlock()
    err := send(client, signature, templateCode, templateParam, phoneNumber)
    if !retry || err == nil {
        return err
    }
	self.initClient()
	// try again
	self.clientLock.RLock()
	client = self.client
	self.clientLock.RUnlock()
    return send(client, signature, templateCode, templateParam, phoneNumber)
}

func send(client *sdk.Client, signature, templateCode, templateParam, phoneNumber string) error {
	m := parser.FindStringSubmatch(phoneNumber)
	if len(m) > 0 {
		if m[1] == "86" {
			phoneNumber = m[2]
		} else {
			phoneNumber = m[1] + m[2]
		}
	}
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

	return checkResponseAndError(client.ProcessCommonRequest(request))
}

func checkResponseAndError(rep *responses.CommonResponse, err error) error {
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
		Code    string
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
