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

package common

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/notify-plugin/pkg/apis"
)

var ErrCodeMap = make(map[error]codes.Code)

func RegisterErr(originErr error, errCode codes.Code) {
	ErrCodeMap[originErr] = errCode
}

func ConvertErr(err error) error {
	if err == nil {
		return nil
	}
	if code, ok := ErrCodeMap[errors.Cause(err)]; ok {
		return status.Error(code, err.Error())
	}
	return status.Error(codes.Internal, err.Error())
}

type Server struct {
	Sender ISender
}

func NewServer(sender ISender) *Server {
	return &Server{Sender:sender}
}

func (s *Server) Send(ctx context.Context, req *apis.SendParams) (*apis.Empty, error) {
	empty := &apis.Empty{}
	if !s.Sender.IsReady(ctx) {
		return empty, status.Error(codes.FailedPrecondition, NOTINIT)
	}
	log.Debugf("recevie msg, contact: %s, title: %s, content: %s", req.Contact, req.Title, req.Message)
	err := s.Sender.Send(ctx, req)
	return empty, ConvertErr(err)
}

func (s *Server) UpdateConfig(ctx context.Context, req *apis.UpdateConfigParams) (empty *apis.Empty, err error) {
	empty = new(apis.Empty)
	defer func() {
		if err != nil {
			log.Errorf("update config error: %s", err.Error())
		}
	}()
	if req.Configs == nil {
		return empty, status.Error(codes.InvalidArgument, ConfigNil)
	}
	log.Debugf("update configs: %v", req.Configs)
	err = s.Sender.UpdateConfig(ctx, req.Configs)
	return empty, ConvertErr(err)
}

func (s *Server) ValidateConfig(ctx context.Context, req *apis.UpdateConfigParams) (*apis.ValidateConfigReply, error) {
	rep := &apis.ValidateConfigReply{}
	if req.Configs == nil {
		return rep, status.Error(codes.InvalidArgument, ConfigNil)
	}
	log.Debugf("validate configs: %v", req.Configs)
	formatConfig, err := s.Sender.CheckConfig(ctx, req.Configs)
	if err != nil {
		log.Errorf(err.Error())
		return rep, status.Error(codes.InvalidArgument, err.Error())
	}
	rep.IsValid, rep.Msg, err = s.Sender.ValidateConfig(ctx, formatConfig)
	if err != nil {
		log.Errorf(err.Error())
		return rep, ConvertErr(err)
	}
	return rep, nil
}

func (s *Server) UseridByMobile(ctx context.Context, req *apis.UseridByMobileParams) (*apis.UseridByMobileReply, error) {
	rep := &apis.UseridByMobileReply{}
	if !s.Sender.IsReady(ctx) {
		return rep, status.Error(codes.FailedPrecondition, NOTINIT)
	}
	log.Debugf("fetch userid by mobile %s", req.Mobile)
	userId, err := s.Sender.FetchContact(ctx, req.Mobile)
	if err != nil {
		return rep, ConvertErr(err)
	}
	rep.Userid = userId
	return rep, nil
}
