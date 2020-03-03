package feishu

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/notify-plugin/common"
	"yunion.io/x/notify-plugin/pkg/apis"
)

type Server struct {
	apis.UnimplementedSendAgentServer
}

func (s *Server) Send(ctx context.Context, req *apis.SendParams) (*apis.Empty, error) {
	log.Debugf("reviced send request")
	empty := &apis.Empty{}
	if sendManager.client == nil {
		return empty, status.Error(codes.FailedPrecondition, common.NOTINIT)
	}
	log.Debugf("reviced msg for %s: %s", req.Contact, req.Message)
	sendManager.workerChan <- struct{}{}
	err := sendManager.send(req)
	<-sendManager.workerChan
	log.Debugf("send over")
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
	sendManager.configCache.BatchSet(req.Configs)
	err = sendManager.initClient()
	if errors.Cause(err) == common.ErrConfigMiss {
		return empty, status.Error(codes.FailedPrecondition, err.Error())
	}
	if err != nil {
		return empty, status.Error(codes.Unavailable, err.Error())
	}
	return
}

func (s *Server) UseridByMobile(ctx context.Context, req *apis.UseridByMobileParams) (*apis.UseridByMobileReply, error) {
	reply := new(apis.UseridByMobileReply)

	if sendManager.client == nil {
		return reply, status.Error(codes.FailedPrecondition, common.NOTINIT)
	}
	userId, err := sendManager.userIdByMobile(req.Mobile)
	if err != nil {
		if errors.Cause(err) == errors.ErrNotFound {
			return reply, status.Error(codes.NotFound, err.Error())
		}
		return reply, status.Error(codes.Internal, err.Error())
	}
	reply.Userid = userId
	return reply, nil
}
