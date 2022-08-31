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

package aliyun

import (
	"encoding/json"
	"regexp"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	sdkerrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/notify-plugins/pkg/sms/driver"
)

var parser = regexp.MustCompile(`\+(\d*) (.*)`)

func sendSms(accessKeyID, accessKeySecret string, signature, templateCode string, templateParam map[string]string, phoneNumber string) error {
	log.Debugf("sendSms: accessKeyID=%s accessKeySecret=%s signature=%s templateCode=%s templateParam=%s phoneNumber=%s", accessKeyID, accessKeySecret, signature, templateCode, templateParam, phoneNumber)
	// lock and update
	client, err := sdk.NewClientWithAccessKey("default", accessKeyID, accessKeySecret)
	if err != nil {
		return err
	}

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
	param, _ := json.Marshal(templateParam)
	request.QueryParams["TemplateParam"] = string(param)

	return checkResponseAndError(client.ProcessCommonRequest(request))
}

func checkResponseAndError(rep *responses.CommonResponse, err error) error {
	if err != nil {
		log.Errorf("aliyun sms send error: %s", jsonutils.Marshal(err))
		serr, ok := err.(*sdkerrors.ServerError)
		if !ok {
			return err
		}
		if serr.ErrorCode() == ACCESSKEYID_NOTFOUND {
			return driver.ErrAccessKeyIdNotFound
		}
		if serr.ErrorCode() == SIGN_DOESNOTMATCH {
			return driver.ErrSignatureDoesNotMatch
		}
		return err
	}

	type RepContent struct {
		Message string
		Code    string
	}
	respContent := rep.GetHttpContentBytes()
	log.Debugf("resp: %s", string(respContent))
	rc := RepContent{}
	err = json.Unmarshal(respContent, &rc)
	if err != nil {
		log.Errorf("The Response Content's style may changed")
		return errors.Wrap(err, "json.Unmarshal")
	}
	if rc.Code == "OK" {
		return nil
	}
	if rc.Code == SIGHNTURE_ILLEGAL {
		return driver.ErrSignnameInvalid
	} else if rc.Code == TEMPLATE_ILLGAL {
		return driver.ErrSignnameInvalid
	}
	return errors.Error(rc.Message)
}
