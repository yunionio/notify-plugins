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
	"fmt"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/notify-plugins/pkg/apis"
)

var (
	ErrNoSuchMobile     = errors.Error("No such mobile")
	ErrIncompleteConfig = errors.Error("Incomplete config")
	ErrDuplicateConfig  = errors.Error("Duplicate config for a domain")
)

func init() {
	RegisterErr(ErrNoSuchMobile, codes.NotFound)
	RegisterErr(ErrIncompleteConfig, codes.PermissionDenied)
	RegisterErr(ErrDuplicateConfig, codes.AlreadyExists)
}

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
	st := status.New(codes.Internal, "test")
	st.WithDetails()
	return status.Error(codes.Internal, err.Error())
}

type ValidateConfig func(ctx context.Context, configs map[string]string) (bool, string, error)

type Server struct {
	senderGenerator func() ISender
	senders         *sync.Map
	validateConfig  ValidateConfig
	senderWapper    SenderWapper
}

func NewServer(senderGenerator func() ISender, validateConfig ValidateConfig, wapper SenderWapper) *Server {
	if wapper == nil {
		wapper = sender
	}
	return &Server{
		senderGenerator: senderGenerator,
		senders:         &sync.Map{},
		validateConfig:  validateConfig,
		senderWapper:    wapper,
	}
}

func sender(domainId string, senders *sync.Map) (ISender, bool) {
	obj, ok := senders.Load(domainId)
	if ok {
		return obj.(ISender), ok
	}
	log.Infof("no sender for domainId %s", domainId)
	// try "" domainId
	obj, ok = senders.Load("")
	if ok {
		return obj.(ISender), ok
	}
	log.Infof("no sender for domainId %s", "")
	return nil, ok
}

func (s *Server) Send(ctx context.Context, req *apis.SendParams) (*apis.Empty, error) {
	if req.Receiver == nil {
		return nil, status.Error(codes.InvalidArgument, "receiver is nil")
	}
	empty := &apis.Empty{}
	sender, ok := s.senderWapper(req.Receiver.DomainId, s.senders)
	if !ok || !sender.IsReady(ctx) {
		return empty, status.Error(codes.FailedPrecondition, fmt.Sprintf("no valid sender for domainId %s", req.Receiver.DomainId))
	}
	log.Debugf("recevie msg, contact: %s, title: %s, content: %s", req.Receiver.Contact, req.Title, req.Message)
	err := sender.Send(ctx, &SendParam{
		Contact:        req.Receiver.Contact,
		Topic:          req.Topic,
		Title:          req.Title,
		Message:        req.Message,
		Priority:       req.Priority,
		RemoteTemplate: req.RemoteTemplate,
	})
	return empty, ConvertErr(err)
}

func (s *Server) Ready(ctx context.Context, req *apis.ReadyInput) (*apis.ReadyOutput, error) {
	for _, domainId := range req.DomainIds {
		if _, ok := s.senders.Load(domainId); !ok {
			return &apis.ReadyOutput{
				Ok: false,
			}, nil
		}
	}
	return &apis.ReadyOutput{
		Ok: true,
	}, nil
}

func (s *Server) appendFailedRecord(records []*apis.FailedRecord, contacts []string, domainId string, reason string) []*apis.FailedRecord {
	for i := range contacts {
		records = append(records, &apis.FailedRecord{
			Receiver: &apis.SReceiver{
				Contact:  contacts[i],
				DomainId: domainId,
			},
			Reason: reason,
		})
	}
	return records
}

func (s *Server) BatchSend(ctx context.Context, req *apis.BatchSendParams) (*apis.BatchSendReply, error) {
	domainContacts := make(map[string][]string)
	for _, rec := range req.Receivers {
		if rec == nil {
			continue
		}
		domainContacts[rec.DomainId] = append(domainContacts[rec.DomainId], rec.Contact)
	}
	log.Infof("req", jsonutils.Marshal(req))
	reply := &apis.BatchSendReply{}
	for domainId, contacts := range domainContacts {
		sender, ok := s.senderWapper(domainId, s.senders)
		if !ok || !sender.IsReady(ctx) {
			reply.FailedRecords = s.appendFailedRecord(reply.FailedRecords, contacts, domainId, fmt.Sprintf("no valid sender for domainId %s", domainId))
			continue
		}
		log.Debugf("recevie msg, contacts: %v, title: %s, content: %s", contacts, req.Title, req.Message)
		records, err := sender.BatchSend(ctx, &BatchSendParam{
			Contacts:       contacts,
			Title:          req.Title,
			Message:        req.Message,
			Priority:       req.Priority,
			RemoteTemplate: req.RemoteTemplate,
		})
		if err != nil {
			reply.FailedRecords = s.appendFailedRecord(reply.FailedRecords, contacts, domainId, err.Error())
			continue
		}
		for _, record := range records {
			reply.FailedRecords = append(reply.FailedRecords, &apis.FailedRecord{
				Receiver: &apis.SReceiver{
					DomainId: domainId,
					Contact:  record.Contact,
				},
				Reason: record.Reason,
			})
		}
	}
	return reply, nil
}

func (s *Server) AddConfig(ctx context.Context, req *apis.AddConfigInput) (empty *apis.Empty, err error) {
	empty = &apis.Empty{}
	domainId := req.DomainId
	_, ok := s.senders.Load(domainId)
	if ok {
		return empty, status.Error(codes.AlreadyExists, fmt.Sprintf("config of domainId %q has been existed", domainId))
	}
	return empty, s.addSender(ctx, domainId, req.Configs)
}

func (s *Server) addSender(ctx context.Context, domainId string, config map[string]string) error {
	sender := s.senderGenerator()
	err := sender.UpdateConfig(ctx, config)
	if err != nil {
		return err
	}
	s.senders.Store(domainId, sender)
	return nil
}

func (s *Server) CompleteConfig(ctx context.Context, req *apis.CompleteConfigInput) (empty *apis.Empty, err error) {
	log.Infof("start to CompleteConfig, req: %s", jsonutils.Marshal(req))
	empty = &apis.Empty{}
	for _, dc := range req.ConfigInput {
		if _, ok := s.senders.Load(dc.DomainId); ok {
			continue
		}
		if err := s.addSender(ctx, dc.DomainId, dc.Configs); err != nil {
			return empty, err
		}
	}
	log.Infof("s.sender: %v", s.senders)
	return empty, nil
}

func (s *Server) DeleteConfig(ctx context.Context, req *apis.DeleteConfigInput) (empty *apis.Empty, err error) {
	empty = &apis.Empty{}
	s.senders.Delete(req.DomainId)
	return empty, nil
}

func (s *Server) UpdateConfig(ctx context.Context, req *apis.UpdateConfigInput) (empty *apis.Empty, err error) {
	empty = &apis.Empty{}
	sender, ok := s.senderWapper(req.DomainId, s.senders)
	if !ok {
		return empty, status.Error(codes.NotFound, fmt.Sprintf("no such config of domainId %q", req.DomainId))
	}
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
	err = sender.UpdateConfig(ctx, req.Configs)
	return empty, ConvertErr(err)
}

func (s *Server) ValidateConfig(ctx context.Context, req *apis.ValidateConfigInput) (*apis.ValidateConfigReply, error) {
	rep := &apis.ValidateConfigReply{}
	if s.validateConfig == nil {
		return rep, errors.ErrNotImplemented
	}
	if req.Configs == nil {
		return rep, status.Error(codes.InvalidArgument, ConfigNil)
	}
	log.Debugf("validate configs: %v", req.Configs)
	var err error
	rep.IsValid, rep.Msg, err = s.validateConfig(ctx, req.Configs)
	if err != nil {
		log.Errorf(err.Error())
		return rep, ConvertErr(err)
	}
	return rep, nil
}

func (s *Server) UseridByMobile(ctx context.Context, req *apis.UseridByMobileParams) (*apis.UseridByMobileReply, error) {
	rep := &apis.UseridByMobileReply{}
	sender, ok := s.senderWapper(req.DomainId, s.senders)
	if !ok || !sender.IsReady(ctx) {
		return rep, status.Error(codes.FailedPrecondition, fmt.Sprintf("No valid sender for domainId %s", req.DomainId))
	}
	log.Debugf("fetch userid by mobile %s", req.Mobile)
	userId, err := sender.FetchContact(ctx, req.Mobile)
	if err != nil {
		return rep, ConvertErr(err)
	}
	rep.Userid = userId
	return rep, nil
}
