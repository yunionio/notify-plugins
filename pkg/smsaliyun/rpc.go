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
	"fmt"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"yunion.io/x/log"

	"notify-plugin/pkg/apis"
)

type Server struct {
	apis.UnimplementedSendAgentServer
	name string
}

func (s *Server) Send(ctx context.Context, req *apis.SendParams) (*apis.Empty, error) {
	empty := &apis.Empty{}
	if senderManager.client == nil {
		return empty, status.Error(codes.FailedPrecondition, NOTINIT)
	}
	request := requests.NewCommonRequest()
	request.Method = "POST"
	request.Scheme = "https" // https | http
	request.Domain = "dysmsapi.aliyuncs.com"
	request.Version = "2017-05-25"
	request.ApiName = "SendSms"
	request.QueryParams["RegionId"] = "default"
	request.QueryParams["PhoneNumbers"] = req.Contact

	signature, _ := senderManager.configCache[SIGNATURE]
	if len(req.RemoteTemplate) == 0 {
		return empty, status.Error(codes.InvalidArgument, NEED_REMOTE_TEMPLATE)
	}
	request.QueryParams["TemplateCode"] = req.RemoteTemplate
	request.QueryParams["TemplateParam"] = req.Message
	// 控制和smsaliyun的最大并发数
	senderManager.workerChan <- struct{}{}
	err := senderManager.send(nil, signature, req.RemoteTemplate, req.Message, req.Contact, true)
	<-senderManager.workerChan
	if err != nil {
		log.Errorf(err.Error())
		return empty, status.Error(codes.Internal, err.Error())
	}
	return empty, nil
}

func (s *Server) UpdateConfig(ctx context.Context, req *apis.UpdateConfigParams) (*apis.Empty, error) {
	empty := &apis.Empty{}
	if req.Configs == nil {
		return empty, status.Error(codes.InvalidArgument, "Config shouldn't be nil")
	}
	senderManager.configLock.Lock()
	shouldInit := false
	for key, value := range req.Configs {
		if key == ACESS_KEY_SECRET_BP {
			key = ACCESS_KEY_SECRET
		}
		if key == ACESS_KEY_ID_BP {
			key = ACCESS_KEY_ID
		}
		if key == ACCESS_KEY_SECRET || key == ACCESS_KEY_ID {
			shouldInit = true
		}
		log.Debugf("update config: %s: %s", key, value)
		senderManager.configCache[key] = value
	}
	senderManager.configLock.Unlock()
	if shouldInit {
		senderManager.initClient()
	}
	return empty, nil
}

func (s *Server) ValidateConfig(ctx context.Context, config *apis.UpdateConfigParams) (*apis.ValidateConfigReply, error) {
	accessKeyId, ok := config.Configs[ACCESS_KEY_ID]
	if !ok {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("require %s", ACCESS_KEY_ID))
	}
	accessKeySecret, ok := config.Configs[ACCESS_KEY_SECRET]
	if !ok {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("require %s", ACCESS_KEY_SECRET))
	}
	signature, ok := config.Configs[SIGNATURE]
	if !ok {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("require %s", SIGNATURE))
	}

	client, err := sdk.NewClientWithAccessKey("default", accessKeyId, accessKeySecret)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("NewClientWithAccessKey: %s", err.Error()))
	}
	rep := apis.ValidateConfigReply{IsValid: true}
	err = senderManager.send(client, signature, "SMS_123456789", `{"code": "123456"}`, "12345678901", false)
	if err == ErrSignnameInvalid || err == ErrSignatureDoesNotMatch || err == ErrAccessKeyIdNotFound {
		rep.IsValid = false
		rep.Msg = err.Error()
		return &rep, nil
	}
	return &rep, nil
}
