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

package workwx

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	wx "github.com/xen0n/go-workwx"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/notify-plugins/pkg/common"
)

type SConnInfo struct {
	CorpID     string
	AgentID    string
	CorpSecret string
}

type SWorkwxSender struct {
	common.SSenderBase
	client     *wx.WorkwxApp
	clientLock sync.Mutex
	stop       context.CancelFunc
}

func (ws *SWorkwxSender) IsReady(ctx context.Context) bool {
	return ws.client != nil
}

func (ws *SWorkwxSender) UpdateConfig(ctx context.Context, configs map[string]string) error {
	ws.ConfigCache.BatchSet(configs)
	return ws.initClient()
}

func ValidateConfig(ctx context.Context, configs map[string]string) (isValid bool, msg string, e error) {
	vals, ok, noKey := common.CheckMap(configs, CORP_ID, CORP_SECRET, AGENT_ID)
	if !ok {
        e = fmt.Errorf("require %s", noKey)
        return
	}
	_, err := strconv.Atoi(vals[2])
	if err != nil {
		e =  fmt.Errorf("the value of %s should be number format", AGENT_ID)
        return
	}
    corpId, corpSecret, agentId := vals[0], vals[1], vals[2]

	// check info.AgentID
	_, err = strconv.Atoi(agentId)
	if err != nil {
		msg = fmt.Sprintf("the value of %s should be number format", AGENT_ID)
		return
	}
	return checkCropIDAndSecret(corpId, corpSecret)
}

func (ws *SWorkwxSender) FetchContact(ctx context.Context, related string) (string, error) {
	userid, err := ws.client.GetUserIDByMobile(related)
	if err == nil {
		return userid, nil
	}
	cErr, iok := err.(*wx.WorkwxClientError)
	if !iok {
		return "", err
	}
	if cErr.Code == 48002 || cErr.Code == 60020 {
		return "", errors.Wrap(common.ErrIncompleteConfig, err.Error())
	}
	if cErr.Code == 60103 || cErr.Code == 46004 {
		return "", errors.Wrap(common.ErrNoSuchMobile, err.Error())
	}
	return userid, cErr
}

func (ws *SWorkwxSender) Send(ctx context.Context, params *common.SendParam) error {
	re := wx.Recipient{
		UserIDs: []string{params.Contact},
	}
	content := fmt.Sprintf("# %s\n\n%s", params.Title, params.Message)
	return ws.client.SendMarkdownMessage(&re, content, false)
}

func (ws *SWorkwxSender) BatchSend(ctx context.Context, params *common.BatchSendParam) ([]*common.FailedRecord, error) {
	re := wx.Recipient{
		UserIDs: params.Contacts,
	}
	content := fmt.Sprintf("# %s\n\n%s", params.Title, params.Message)
	resp, err := ws.client.SendMarkdownMessageWithResp(&re, content, false)
	if err != nil {
		return nil, err
	}
	var invalidUsers []string
	if len(resp.InvalidUsers) > 0 {
		invalidUsers = strings.Split(resp.InvalidUsers, ",")
	}
	records := make([]*common.FailedRecord, len(invalidUsers))
	for i := range records {
		record := &common.FailedRecord{
			Contact: invalidUsers[i],
			Reason:  "invalid user",
		}
		records[i] = record
	}
	return records, nil
}

func NewSender(config common.IServiceOptions) common.ISender {
	return &SWorkwxSender{
		SSenderBase: common.NewSSednerBase(config),
	}
}

func checkCropIDAndSecret(corpId, corpSecret string) (ok bool, msg string, err error) {
	checkApp := wx.New(corpId).WithApp(corpSecret, 0)
	err = checkApp.SyncAccessToken()
	if err == nil {
		ok = true
		return
	}
	ok = false
	cErr, iok := err.(*wx.WorkwxClientError)
	if !iok {
		return
	}
	err = nil
	if cErr.Code == INVALID_CORP_ID {
		msg = "invalid corpid"
		return
	}
	if cErr.Code == INVALID_CORP_SECRET {
		msg = "invalid corpSecret"
		return
	}
	msg = cErr.Msg
	return
}

func (ws *SWorkwxSender) initClient() error {
	vals, ok, noKey := ws.ConfigCache.BatchGet(CORP_ID, CORP_SECRET, AGENT_ID)
	if !ok {
		return errors.Wrap(common.ErrConfigMiss, noKey)
	}
	corpId, corpSecret := vals[0], vals[1]
	agentId, _ := strconv.Atoi(vals[2])

	app := wx.New(corpId).WithApp(corpSecret, int64(agentId))
	ctx, cancel := context.WithCancel(context.Background())
	app.SpawnAccessTokenRefresherWithContext(ctx)

	ws.clientLock.Lock()
	defer ws.clientLock.Unlock()
	// stop previous one
	if ws.stop != nil {
		ws.stop()
	}
	ws.stop = cancel
	ws.client = app
	return nil
}
