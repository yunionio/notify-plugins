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
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"yunion.io/x/log"

	"yunion.io/x/notify-plugin/pkg/apis"
	"yunion.io/x/notify-plugin/common"
)

type Server struct {
	apis.UnimplementedSendAgentServer
}

func (s *Server) Send(ctx context.Context, req *apis.SendParams) (*apis.Empty, error) {
	empty := &apis.Empty{}
	if senderManager.msgChan == nil {
		err := status.Error(codes.FailedPrecondition, common.NOTINIT)
		return empty, err
	}
	log.Debugf("reviced msg for %s: %s", req.Contact, req.Message)
	err := senderManager.send(req)
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
	err = senderManager.restartSender()
	if err != nil {
		return empty, status.Error(codes.FailedPrecondition, err.Error())
	}
	return
}

func (s *Server) ValidateConfig(ctx context.Context, config *apis.UpdateConfigParams) (*apis.ValidateConfigReply, error) {
	vals, ok, noKey := common.CheckMap(config.Configs, HOSTNAME, HOSTPORT, USERNAME, PASSWORD)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("require %s", noKey))
	}
	hostname, hostport, username, password := vals[0], vals[1], vals[2], vals[3]
	port, err := strconv.Atoi(hostport)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid hostport %s", hostport))
	}

	err = senderManager.validateConfig(sConnectInfo{hostname, port, username, password})
	if err == nil {
		return &apis.ValidateConfigReply{IsValid: true, Msg: ""}, nil
	}

	reply := apis.ValidateConfigReply{IsValid: false}

	switch {
	case strings.Contains(err.Error(), "535 Error"):
		reply.Msg = "Authentication failed"
	case strings.Contains(err.Error(), "timeout"):
		reply.Msg = "Connect timeout"
	case strings.Contains(err.Error(), "no such host"):
		reply.Msg = "No such host"
	default:
		reply.Msg = err.Error()
	}

	return &reply, nil
}
