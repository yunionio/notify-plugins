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
	"sync"

	wx "github.com/xen0n/go-workwx"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/notify-plugin/pkg/apis"
	"yunion.io/x/notify-plugin/pkg/common"
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

func (ws *SWorkwxSender) CheckConfig(ctx context.Context, configs map[string]string) (interface{}, error) {
	vals, ok, noKey := common.CheckMap(configs, CORP_ID, CORP_SECRET, AGENT_ID)
	if !ok {
		return nil, fmt.Errorf("require %s", noKey)
	}
	_, err := strconv.Atoi(vals[2])
	if err != nil {
		return nil, fmt.Errorf("the value of %s should be number format", AGENT_ID)
	}
	return SConnInfo{CorpID: vals[0], CorpSecret: vals[1], AgentID: vals[2]}, nil
}

func (ws *SWorkwxSender) UpdateConfig(ctx context.Context, configs map[string]string) error {
	ws.ConfigCache.BatchSet(configs)
	return ws.initClient()
}

func (ws *SWorkwxSender) ValidateConfig(ctx context.Context, configs interface{}) (isValid bool, msg string, e error) {
	info := configs.(SConnInfo)

	// check info.AgentID
	_, err := strconv.Atoi(info.AgentID)
	if err != nil {
		msg = fmt.Sprintf("the value of %s should be number format", AGENT_ID)
		return
	}
	return ws.checkCropIDAndSecret(info.CorpID, info.CorpSecret)
}

func (ws *SWorkwxSender) FetchContact(ctx context.Context, related string) (string, error) {
	userid, err := ws.client.GetUserIDByMobile(related)
	if err != nil {
		return "", err
	}
	return userid, nil
}

func (ws *SWorkwxSender) Send(ctx context.Context, params *apis.SendParams) error {
	re := wx.Recipient{
		UserIDs: []string{params.Contact},
	}
	content := fmt.Sprintf("# %s\n\n%s", params.Title, params.Message)
	return ws.client.SendMarkdownMessage(&re, content, true)
}

func NewSender(config common.IServiceOptions) common.ISender {
	return &SWorkwxSender{
		SSenderBase: common.NewSSednerBase(config),
	}
}

func (ws *SWorkwxSender) checkCropIDAndSecret(corpId, corpSecret string) (ok bool, msg string, err error) {
	checkApp := wx.New(corpId).WithApp(corpSecret, 0)
	err = checkApp.SyncAccessToken()
	if err == nil {
		ok = true
		return
	}
	cErr, ok := err.(*wx.WorkwxClientError)
	if !ok {
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
