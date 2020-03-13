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

	"yunion.io/x/notify-plugin/pkg/common"
	"yunion.io/x/notify-plugin/pkg/apis"
)

type SConnectInfo struct {
	Hostname string
	Hostport int
	Username string
	Password string
}

type SSenderManager struct {
	msgChan    chan *sSendUnit
	senders    []sSender
	senderNum  int
	chanelSize int

	configCache *common.SConfigCache
}

func (self *SSenderManager) IsReady(ctx context.Context) bool {
	return self.msgChan == nil
}

func (self *SSenderManager) CheckConfig(ctx context.Context, configs map[string]string) (interface{}, error) {
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

func (self *SSenderManager) UpdateConfig(ctx context.Context, configs map[string]string) error {
	self.configCache.BatchSet(configs)
	return self.restartSender()
}

func (self *SSenderManager) ValidateConfig(ctx context.Context, configs interface{}) (*apis.ValidateConfigReply, error) {
	connInfo := configs.(SConnectInfo)
	err := self.validateConfig(connInfo)
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

func (self *SSenderManager) FetchContact(ctx context.Context, related string) (string, error) {
	return "", nil
}

func (self *SSenderManager) Send(ctx context.Context, params *apis.SendParams) error {
	log.Debugf("reviced msg for %s: %s", params.Contact, params.Message)
	return senderManager.send(params)
}

func NewSender(config common.IServiceOptions) common.ISender {
	return &SSenderManager{
		senders:    make([]sSender, config.GetSenderNum()),
		senderNum:  config.GetSenderNum(),
		chanelSize: config.GetOthers().(int),

		configCache: common.NewConfigCache(),
	}
}

func (self *SSenderManager) send(args *apis.SendParams) error {
	gmsg := gomail.NewMessage()
	username, _ := senderManager.configCache.Get(USERNAME)
	gmsg.SetHeader("From", username)
	gmsg.SetHeader("To", args.Contact)
	gmsg.SetHeader("Subject", args.Topic)
	gmsg.SetHeader("Subject", args.Title)
	gmsg.SetBody("text/html", args.Message)
	ret := make(chan bool)
	senderManager.msgChan <- &sSendUnit{gmsg, ret}
	if suc := <-ret; !suc {
		// try again
		senderManager.msgChan <- &sSendUnit{gmsg, ret}
		if suc = <-ret; !suc {
			return errors.Error("send error")
		}
	}
	return nil
}

func (self *SSenderManager) restartSender() error {
	for _, sender := range self.senders {
		sender.stop()
	}
	return self.initSender()
}

func (self *SSenderManager) validateConfig(connInfo SConnectInfo) error {
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

func (self *SSenderManager) initSender() error {
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
}

func (self *sSender) Run() {
	var err error
Loop:
	for {
		select {
		case msg, ok := <-senderManager.msgChan:
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
