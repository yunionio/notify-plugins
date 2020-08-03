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
	"yunion.io/x/notify-plugin/pkg/apis"
	"yunion.io/x/notify-plugin/pkg/common"
	"yunion.io/x/pkg/errors"
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

func (ws *SWorkwxSender) ValidateConfig(ctx context.Context, configs interface{}) (bool, string, error) {
	panic("not implemented") // TODO: Implement
}

func (ws *SWorkwxSender) FetchContact(ctx context.Context, related string) (string, error) {
	panic("not implemented") // TODO: Implement
}

func (ws *SWorkwxSender) Send(ctx context.Context, params *apis.SendParams) error {
	panic("not implemented") // TODO: Implement
}

func (ws *SWorkwxSender) initClient() error {
	vals, ok, noKey := ws.ConfigCache.BatchGet(CORP_ID, CORP_SECRET, AGENT_ID)
	if !ok {
		return errors.Wrap(common.ErrConfigMiss, noKey)
	}
	corpId, corpSecret := vals[0], vals[1]
	agentId, _ := strconv.Atoi(vals[2])

	app := wx.New(corpId).WithApp(corpSecret, int64(agentId))
	err := app.SyncAccessToken()
	if err != nil {
		return nil
	}
	app.SpawnAccessTokenRefresher()
	ws.clientLock.Lock()
	defer ws.clientLock.Unlock()
	ws.client = app
	return nil
}
