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
	"yunion.io/x/notify-plugin/common"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"yunion.io/x/log"

	"yunion.io/x/notify-plugin/pkg/apis"
)

type Server struct {
	apis.UnimplementedSendAgentServer
}

func (s *Server) Send(ctx context.Context, req *apis.SendParams) (*apis.Empty, error) {
	empty := &apis.Empty{}
	if senderManager.session == nil {
		return empty, status.Error(codes.FailedPrecondition, common.NOTINIT)
	}
	senderManager.workerChan <- struct{}{}
	err := senderManager.send(req)
	<-senderManager.workerChan
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
	senderManager.configCache.BatchSet(req.Configs)
	err = senderManager.initClient()
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	return
}
