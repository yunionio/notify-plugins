package feishu

import (
	"sync"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/monitor/notifydrivers/feishu"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/notify-plugin/common"
	"yunion.io/x/notify-plugin/pkg/apis"
)

type sSendManager struct {
	workerChan chan struct{}
	client     *feishu.Tenant
	clientLock sync.RWMutex

	configCache *common.SConfigCache
}

func newSSendManager(config *common.SBaseOptions) *sSendManager {
	log.Debugf("sender num: %d", config.SenderNum)
	return &sSendManager{
		workerChan:  make(chan struct{}, config.SenderNum),
		configCache: common.NewConfigCache(),
	}
}

func (self *sSendManager) send(args *apis.SendParams) error {
	req := feishu.MsgReq{
		OpenId:  args.Contact,
		MsgType: "text",
		Content: &feishu.MsgContent{Text: args.Message},
	}
	_, err := self.client.SendMessage(req)
	if err != nil {
		err = errors.Wrap(err, "SendMessage")
	}
	return err
}

func (self *sSendManager) initClient() error {
	vals, ok, noKey := self.configCache.BatchGet(APP_ID, APP_SECRET)
	if !ok {
		return errors.Wrap(common.ErrConfigMiss, noKey)
	}
	appID, appSecret := vals[0], vals[1]

	// lock and update
	client, err := feishu.NewTenant(appID, appSecret)
	if err != nil {
		return err
	}
	self.clientLock.Lock()
	defer self.clientLock.Unlock()
	self.client = client
	return nil
}

func (self *sSendManager) userIdByMobile(mobile string) (string, error) {
	return self.client.UserIdByMobile(mobile)
}
