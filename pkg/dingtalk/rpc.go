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
	"yunion.io/x/pkg/errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"yunion.io/x/log"

	"yunion.io/x/notify-plugin/common"
	"yunion.io/x/notify-plugin/pkg/apis"
)

type Server struct {
	apis.UnimplementedSendAgentServer
}

func (s *Server) Send(ctx context.Context, req *apis.SendParams) (*apis.Empty, error) {
	empty := &apis.Empty{}
	if senderManager.client == nil {
		err := status.Error(codes.FailedPrecondition, common.NOTINIT)
		return empty, err
	}
	sendFunc := senderManager.getSendFunc(req)
	senderManager.workerChan <- struct{}{}
	err := senderManager.send(sendFunc)
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

func (s *Server) UpdateConfig(ctx context.Context, req *apis.UpdateConfigParams) (empty *apis.Empty, err error) {
	defer func() {
		if err != nil {
			log.Errorf(err.Error())
		}
	}()
	empty = new(apis.Empty)
	if req.Configs == nil {
		return empty, status.Error(codes.InvalidArgument, common.ConfigNil)
	}
	log.Debugf("update config...")
	senderManager.configCache.BatchSet(req.Configs)
	err = senderManager.initClient()
	if errors.Cause(err) == common.ErrConfigMiss {
		return empty, status.Error(codes.FailedPrecondition, err.Error())
	}
	if err != nil {
		return empty, status.Error(codes.Internal, err.Error())
	}
	return
}

func (s *Server) UseridByMobile(ctx context.Context, req *apis.UseridByMobileParams) (*apis.UseridByMobileReply, error) {
	reply := &apis.UseridByMobileReply{}

	if senderManager.client == nil {
		err := status.Error(codes.FailedPrecondition, common.NOTINIT)
		return reply, err
	}
	userId, err := senderManager.getUseridByMobile(req.Mobile)
	if err == ErrNoSuchUser {
		return reply, status.Error(codes.NotFound, err.Error())
	}
	if err != nil {
		return reply, status.Error(codes.Internal, err.Error())
	}
	reply.Userid = userId
	return reply, nil
}
