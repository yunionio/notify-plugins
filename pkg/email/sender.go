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
	"strconv"
	"time"

	"gopkg.in/gomail.v2"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/notify-plugin/pkg/apis"
	"yunion.io/x/notify-plugin/common"
)

type sConnectInfo struct {
	Hostname string
	Hostport int
	Username string
	Password string
}

type sSenderManager struct {
	msgChan     chan *sSendUnit
	senders     []sSender
	senderNum   int
	chanelSize  int

	configCache *common.SConfigCache
}

func newSSenderManager(config *SEmailConfig) *sSenderManager {
	return &sSenderManager{
		senders:    make([]sSender, config.SenderNum),
		senderNum:  config.SenderNum,
		chanelSize: config.ChannelSize,

		configCache: common.NewConfigCache(),
	}
}

func (self *sSenderManager) send(args *apis.SendParams) error {
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

func (self *sSenderManager) restartSender() error {
	for _, sender := range self.senders {
		sender.stop()
	}
	return self.initSender()
}

func (self *sSenderManager) validateConfig(connInfo sConnectInfo) error {
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

func (self *sSenderManager) initSender() error {
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
