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
	if senderManager.msgChan == nil {
		err := status.Error(codes.FailedPrecondition, NOTINIT)
		return empty, err
	}
	log.Debugf("reviced msg for %s: %s", req.Contact, req.Message)
	err := senderManager.send(req)
	if err == ErrTemplate {
		return empty, status.Error(codes.Internal, err.Error())
	}
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
	for key, value := range req.Configs {
		senderManager.configCache[key] = value
	}
	senderManager.configLock.Unlock()
	senderManager.restartSender()
	return empty, nil
}
