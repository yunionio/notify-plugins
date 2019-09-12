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
	"notify-plugin/pkg/apis"
)

type Server struct {
	apis.UnimplementedSendAgentServer
	name string
}

func (s *Server) Send(ctx context.Context, req *apis.SendParams) (*apis.BaseReply, error) {
	reply := &apis.BaseReply{}
	if senderManager.client == nil {
		reply.Success = false
		reply.Msg = NOTINIT
		return reply, nil
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
		reply.Success = false
		reply.Msg = "Corresponding template not found."
		go senderManager.updateTemplateCache()
		return reply, nil
	}
	request.QueryParams["TemplateCode"] = tem
	request.QueryParams["TemplateParam"] = req.Message
	// 控制和smsaliyun的最大并发数
	senderManager.workerChan <- struct{}{}
	senderManager.send(request, reply)
	<-senderManager.workerChan
	return reply, nil
}

func (s *Server) UpdateConfig(ctx context.Context, req *apis.UpdateConfigParams) (*apis.BaseReply, error) {
	reply := &apis.BaseReply{}
	if req.Configs == nil {
		reply.Success = false
		reply.Msg = "Config shouldn't be nil."
		return reply, nil
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
	reply.Success = true
	return reply, nil
}
