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
	"time"

	"gopkg.in/gomail.v2"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/notify-plugin/pkg/apis"
	"yunion.io/x/notify-plugin/pkg/common"
)

type SConnectInfo struct {
	Hostname string
	Hostport int
	Username string
	Password string
}

type SEmailSender struct {
	msgChan    chan *sSendUnit
	senders    []sSender
	senderNum  int
	chanelSize int

	configCache *common.SConfigCache
}

func (self *SEmailSender) IsReady(ctx context.Context) bool {
	return self.msgChan != nil
}

func (self *SEmailSender) CheckConfig(ctx context.Context, configs map[string]string) (interface{}, error) {
	vals, ok, noKey := common.CheckMap(configs, HOSTNAME, HOSTPORT, USERNAME, PASSWORD)
	if !ok {
		return nil, fmt.Errorf("require %s", noKey)
	}

	port, err := strconv.Atoi(vals[1])
	if err != nil {
		return nil, fmt.Errorf("invalid hostport %s", vals[1])
	}
	return SConnectInfo{
		Hostname: vals[0],
		Hostport: port,
		Username: vals[2],
		Password: vals[3],
	}, nil
}

func (self *SEmailSender) UpdateConfig(ctx context.Context, configs map[string]string) error {
	self.configCache.BatchSet(configs)
	return self.restartSender()
}

func (self *SEmailSender) ValidateConfig(ctx context.Context, configs interface{}) (isValid bool, msg string, err error) {
	connInfo := configs.(SConnectInfo)
	err = self.validateConfig(connInfo)
	if err == nil {
		isValid = true
		return
	}

	switch {
	case strings.Contains(err.Error(), "535 Error"):
		msg = "Authentication failed"
	case strings.Contains(err.Error(), "timeout"):
		msg = "Connect timeout"
	case strings.Contains(err.Error(), "no such host"):
		msg = "No such host"
	default:
		msg = err.Error()
	}
	return
}

func (self *SEmailSender) FetchContact(ctx context.Context, related string) (string, error) {
	return "", nil
}

func (self *SEmailSender) Send(ctx context.Context, params *apis.SendParams) error {
	log.Debugf("reviced msg for %s: %s", params.Contact, params.Message)
	return self.send(params)
}

func (self *SEmailSender) BatchSend(ctx context.Context, params *apis.BatchSendParams) ([]*apis.FailedRecord, error) {
	ret := make([]*apis.FailedRecord, len(params.Contacts))
	send := func(i int) {
		param := apis.SendParams{
			Contact:        params.Contacts[i],
			Topic:          params.Title,
			Title:          params.Title,
			Message:        params.Message,
			Priority:       params.Priority,
			RemoteTemplate: params.RemoteTemplate,
		}
		err := self.send(&param)
		if err == nil {
			return
		}
		record := &apis.FailedRecord{
			Contact: params.Contacts[i],
			Reason:  err.Error(),
		}
		ret[i] = record
	}
	for i := range ret {
		send(i)
	}
	// remove nil
	processedRet := make([]*apis.FailedRecord, 0, len(ret))
	for i := range ret {
		if ret[i] == nil {
			continue
		}
		processedRet = append(processedRet, ret[i])
	}
	return processedRet, nil
}

func NewSender(config common.IServiceOptions) common.ISender {
	return &SEmailSender{
		senders:    make([]sSender, config.GetSenderNum()),
		senderNum:  config.GetSenderNum(),
		chanelSize: config.GetOthers().(int),

		configCache: common.NewConfigCache(),
	}
}

func (self *SEmailSender) send(args *apis.SendParams) error {
	gmsg := gomail.NewMessage()
	username, _ := self.configCache.Get(USERNAME)
	gmsg.SetHeader("From", username)
	gmsg.SetHeader("To", args.Contact)
	gmsg.SetHeader("Subject", args.Topic)
	gmsg.SetHeader("Subject", args.Title)
	gmsg.SetBody("text/html", args.Message)
	ret := make(chan bool, 1)
	self.msgChan <- &sSendUnit{gmsg, ret}
	timer := time.NewTimer(1 * time.Minute)
	defer timer.Stop()
	select {
	case suc := <-ret:
		if !suc {
			return errors.Error("send error")
		}
	case <-timer.C:
		return errors.Error("send error, time out")
	}
	return nil
}

func (self *SEmailSender) restartSender() error {
	for _, sender := range self.senders {
		sender.stop()
	}
	return self.initSender()
}

func (self *SEmailSender) validateConfig(connInfo SConnectInfo) error {
	errChan := make(chan error)
	go func() {
		sender, err := gomail.NewDialer(connInfo.Hostname, connInfo.Hostport, connInfo.Username, connInfo.Password).Dial()
		if err != nil {
			errChan <- err
			return
		}
		sender.Close()
		errChan <- nil
	}()

	ticker := time.Tick(5 * time.Second)
	select {
	case <-ticker:
		return errors.Error("535 Error")
	case err := <-errChan:
		return err
	}
}

func (self *SEmailSender) initSender() error {
	vals, ok, noKey := self.configCache.BatchGet(HOSTNAME, PASSWORD, USERNAME, HOSTPORT)
	if !ok {
		return errors.Wrap(common.ErrConfigMiss, noKey)
	}
	hostName, password, userName, hostPortStr := vals[0], vals[1], vals[2], vals[3]
	hostPort, _ := strconv.Atoi(hostPortStr)
	dialer := gomail.NewDialer(hostName, hostPort, userName, password)
	// Configs are obtained successfully, it's time to init msgChan.
	if self.msgChan == nil {
		self.msgChan = make(chan *sSendUnit, self.chanelSize)
	}
	for i := 0; i < self.senderNum; i++ {
		sender := sSender{
			number: i + 1,
			dialer: dialer,
			sender: nil,
			open:   false,
			stopC:  make(chan struct{}),
			man:    self,
		}
		self.senders[i] = sender
		go sender.Run()
	}

	log.Infof("Total %d senders.", self.senderNum)
	return nil
}

type sSender struct {
	number int
	dialer *gomail.Dialer
	sender gomail.SendCloser
	open   bool
	stopC  chan struct{}
	man    *SEmailSender

	closeFailedTimes int
}

func (self *sSender) Run() {
	var err error
Loop:
	for {
		select {
		case msg, ok := <-self.man.msgChan:
			if !ok {
				break Loop
			}
			if !self.open {
				if self.sender, err = self.dialer.Dial(); err != nil {
					log.Errorf("No.%d sender connect to email serve failed because that %s.", self.number, err.Error())
					msg.result <- false
					continue Loop
				}
				self.open = true
				if err := gomail.Send(self.sender, msg.message); err != nil {
					log.Errorf("No.%d sender send email failed because that %s.", self.number, err.Error())
					self.open = false
					msg.result <- false
					continue Loop
				}
				log.Debugf("No.%d sender send email successfully.", self.number)
				msg.result <- true
			}
		case <-self.stopC:
			break Loop
		case <-time.After(30 * time.Second):
			if self.open {
				if err = self.sender.Close(); err != nil {
					log.Errorf("No.%d sender has be idle for 30 seconds and closed failed because that %s.", self.number, err.Error())
					if self.closeFailedTimes > 2 {
						log.Infof("No.%d sender has close failed 2 times so set open as false", self.number)
						self.closeFailedTimes = 0
						self.open = false
					} else {
						self.closeFailedTimes++
					}
					continue Loop
				}
				self.open = false
				log.Infof("No.%d sender has be idle for 30 seconds so that closed temporarily.", self.number)
			}
		}
	}
}

func (self *sSender) stop() {
	// First restart
	if self.stopC == nil {
		return
	}
	close(self.stopC)
}

type sSendUnit struct {
	message *gomail.Message
	result  chan<- bool
}
