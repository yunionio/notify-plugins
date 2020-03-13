package common

import (
	"context"
	"google.golang.org/grpc/codes"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/notify-plugin/pkg/apis"
)

type ISender interface {
	IsReady(ctx context.Context) bool
	CheckConfig(ctx context.Context, configs map[string]string) (interface{}, error)
	UpdateConfig(ctx context.Context, configs map[string]string) error
	ValidateConfig(ctx context.Context, configs interface{}) (*apis.ValidateConfigReply, error)
	FetchContact(ctx context.Context, related string) (string, error)
	Send(ctx context.Context, params *apis.SendParams) error
}

type SSenderBase struct {
	ConfigCache *SConfigCache
	workerChan chan struct{}
}

func init() {
	RegisterErr(errors.ErrNotImplemented, codes.Unimplemented)
}

func (self *SSenderBase) Do(f func() error) error {
	self.workerChan<- struct{}{}
	defer func() {
		<- self.workerChan
	}()
	return f()
}

func (self *SSenderBase) IsReady(ctx context.Context) bool {
	return true
}

func (self *SSenderBase) CheckConfig(ctx context.Context, configs map[string]string) (interface{}, error) {
	return nil, errors.ErrNotImplemented
}

func (self *SSenderBase) UpdateConfig(ctx context.Context, configs map[string]string) error {
	return errors.ErrNotImplemented
}

func (self *SSenderBase) ValidateConfig(ctx context.Context, configs interface{}) (*apis.ValidateConfigReply, error) {
	return nil, errors.ErrNotImplemented
}

func (self *SSenderBase) FetchContact(ctx context.Context, related string) (string, error) {
	return "", errors.ErrNotImplemented
}

func (self *SSenderBase) Send(ctx context.Context, params *apis.SendParams) error {
	return errors.ErrNotImplemented
}

func NewSSednerBase(config IServiceOptions) SSenderBase {
	return SSenderBase{
		ConfigCache: NewConfigCache(),
		workerChan:  make(chan struct{}, config.GetSenderNum()),
	}
}

