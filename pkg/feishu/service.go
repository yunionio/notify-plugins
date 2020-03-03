package feishu

import (
	"yunion.io/x/notify-plugin/common"
	"yunion.io/x/notify-plugin/pkg/apis"
)

var sendManager *sSendManager

func StartService() {
	var config common.SBaseOptions
	common.StartService(&config, &Server{apis.UnimplementedSendAgentServer{}},
		"feishu", "feishu.conf",
		func() {
			sendManager = newSSendManager(&config)
		})
}
