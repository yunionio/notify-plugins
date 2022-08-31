package driver

import (
	"context"

	"yunion.io/x/log"
)

const (
	DriverKey = "sms_driver"

	DriverAliyun = "smsaliyun"
	DriverHuawei = "smshuawei"
)

type ISmsDriver interface {
	Name() string
	Verify(ctx context.Context, config map[string]string) error
	Send(ctx context.Context, config map[string]string, dest string, templateId string, params [][]string) error
}

var (
	smsDriverTable = make(map[string]ISmsDriver)
)

func Register(drv ISmsDriver) {
	smsDriverTable[drv.Name()] = drv
	log.Infof("sms driver %s registered!", drv.Name())
}

func GetDriver(conf map[string]string) ISmsDriver {
	var drvName string
	if drvTmp, ok := conf[DriverKey]; !ok {
		drvName = DriverAliyun
	} else {
		drvName = drvTmp
	}
	return smsDriverTable[drvName]
}
