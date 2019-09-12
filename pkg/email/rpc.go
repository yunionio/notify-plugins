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

package email

import (
	"context"
	"notify-plugin/pkg/apis"
	"yunion.io/x/log"
)

type Server struct {
	apis.UnimplementedSendAgentServer
	name string
}

func (s *Server) Send(ctx context.Context, req *apis.SendParams) (*apis.BaseReply, error) {
	reply := &apis.BaseReply{}
	if senderManager.msgChan == nil {
		reply.Success = false
		reply.Msg = NOTINIT
		return reply, nil
	}
	log.Debugf("reviced msg for %s: %s", req.Contact, req.Message)
	senderManager.send(req, reply)
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
	for key, value := range req.Configs {
		senderManager.configCache[key] = value
	}
	senderManager.configLock.Unlock()
	senderManager.restartSender()
	reply.Success = true
	return reply, nil
}
