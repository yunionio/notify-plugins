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
	"notify-plugin/pkg/apis"
	"yunion.io/x/log"
)

type Server struct {
	name string
}

func (s *Server) Send(ctx context.Context, req *apis.SendParams) (*apis.BaseReply, error) {
	response := apis.BaseReply{}
	if senderManager.client == nil {
		response.Success = false
		response.Msg = NOTINIT
		return &response, nil
	}
	sendFunc, err := senderManager.getSendFunc(req)
	if err != nil {
		response.Success = false
		response.Msg = err.Error()
		return &response, nil
	}

	senderManager.workerChan <- struct{}{}
	senderManager.send(&response, sendFunc)
	<-senderManager.workerChan
	return &response, nil
}

func (s *Server) UpdateConfig(ctx context.Context, req *apis.UpdateConfigParams) (*apis.BaseReply, error) {
	reply := apis.BaseReply{}
	if req.Configs == nil {
		reply.Success = false
		reply.Msg = "Config shouldn't be nil."
		return &reply, nil
	}
	log.Debugf("update config...")
	senderManager.configLock.Lock()
	shouldInit := false
	for key, value := range req.Configs {
		if key == APP_KEY || key == APP_SECRET {
			shouldInit = true
		}
		senderManager.configCache[key] = value
	}
	senderManager.configLock.Unlock()
	if shouldInit {
		senderManager.initClient()
	}
	reply.Success = true
	return &reply, nil
}

func (s *Server) UseridByMobile(ctx context.Context, req *apis.UseridByMobileParams) (*apis.UseridByMobileReply,
	error) {

	userId, err := senderManager.client.UseridByMobile(req.Mobile)
	reply := apis.UseridByMobileReply{}
	reply.Userid = userId
	return &reply, err
}
