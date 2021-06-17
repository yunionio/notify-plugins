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

package robot

import (
	"context"

	"google.golang.org/grpc/codes"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/notify-plugin/pkg/common"
)

const WEBHOOK = "webhook"

var ErrNoSuchWebhook = errors.Error("No such webhook")
var InvalidWebhook = errors.Error("Invalid webhook")

func init() {
	common.RegisterErr(InvalidWebhook, codes.InvalidArgument)
}

type SendFunc func(ctx context.Context, webhook, title, msg string) error

type SRobotSender struct {
	common.SSenderBase
	send SendFunc
}

func NewSender(config common.IServiceOptions, send SendFunc) common.ISender {
	return &SRobotSender{
		SSenderBase: common.NewSSednerBase(config),
		send:        send,
	}
}

func (self *SRobotSender) IsReady(ctx context.Context) bool {
	return true
}

func (self *SRobotSender) Send(ctx context.Context, params *common.SendParam) error {
	return self.send(ctx, params.Contact, params.Title, params.Message)
}

func (self *SRobotSender) BatchSend(ctx context.Context, params *common.BatchSendParam) ([]*common.FailedRecord, error) {
	failedRecords := make([]*common.FailedRecord, 0)
	for _, webhook := range params.Contacts {
		err := self.send(ctx, webhook, params.Title, params.Message)
		if err != nil {
			failedRecords = append(failedRecords, &common.FailedRecord{
				Contact: webhook,
				Reason:  err.Error(),
			})
		}
	}
	return failedRecords, nil
}
