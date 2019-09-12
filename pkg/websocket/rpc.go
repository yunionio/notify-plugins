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

package websocket

import (
	"context"
	"notify-plugin/pkg/apis"
)

type Server struct {
	apis.UnimplementedSendAgentServer
	name string
}

func (s *Server) Send(ctx context.Context, req *apis.SendParams) (*apis.BaseReply, error) {
	reply := &apis.BaseReply{}
	if senderManager.session == nil {
		reply.Success = false
		reply.Msg = NOTINIT
		return reply, nil
	}
	senderManager.workerChan <- struct{}{}
	senderManager.send(req, reply)
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
		if key == AUTH_URI || key == ADMIN_USER || key == ADMIN_PASSWORD {
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
