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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"yunion.io/x/log"

	"notify-plugin/pkg/apis"
)

type Server struct {
	name string
}

func (s *Server) Send(ctx context.Context, req *apis.SendParams) (*apis.Empty, error) {
	empty := &apis.Empty{}
	if senderManager.client == nil {
		err := status.Error(codes.FailedPrecondition, NOTINIT)
		return empty, err
	}
	sendFunc, err := senderManager.getSendFunc(req)
	if err != nil {
		return empty, status.Error(codes.Internal, err.Error())
	}

	senderManager.workerChan <- struct{}{}
	err = senderManager.send(sendFunc)
	<-senderManager.workerChan
	if err == ErrAgentIDNotInit {
		return empty, status.Error(codes.FailedPrecondition, err.Error())
	}
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
	return empty, nil
}

func (s *Server) UseridByMobile(ctx context.Context, req *apis.UseridByMobileParams) (*apis.UseridByMobileReply, error) {
	reply := &apis.UseridByMobileReply{}

	if senderManager.client == nil {
		err := status.Error(codes.FailedPrecondition, NOTINIT)
		return reply, err
	}

	userId, err := senderManager.client.UseridByMobile(req.Mobile)
	reply.Userid = userId
	return reply, status.Error(codes.Internal, err.Error())
}
