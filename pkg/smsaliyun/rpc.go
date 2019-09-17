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

	// Do not need to lock because that do not need signature when re-connect.
	if signature, ok := senderManager.configCache[SIGNATURE]; ok {
		request.QueryParams["SignName"] = signature
	}
	senderManager.templateLock.RLock()
	tem, ok := senderManager.templateCache[req.Topic]
	senderManager.templateLock.RUnlock()
	if !ok {
		err := status.Error(codes.Internal, "Corresponding template not found")
		go senderManager.updateTemplateCache()
		return empty, err
	}
	request.QueryParams["TemplateCode"] = tem
	request.QueryParams["TemplateParam"] = req.Message
	// 控制和smsaliyun的最大并发数
	senderManager.workerChan <- struct{}{}
	err := senderManager.send(request)
	<-senderManager.workerChan
	if err != nil {
		log.Errorf(err.Error())
		return empty, status.Error(codes.Unavailable, err.Error())
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
		senderManager.configCache[key] = value
	}
	senderManager.configLock.Unlock()
	if shouldInit {
		senderManager.initClient()
	}
	return empty, nil
}
