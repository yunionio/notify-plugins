package utils

import (
	"fmt"
	"os"
	"path"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/structarg"
)

func ParseOptions(optStruct interface{}, args []string, configFileName string) {
	parser, err := structarg.NewArgumentParser(optStruct,
		"email-sender", "", "")
	if err != nil {
		log.Fatalf("Error define argument parser: %v", err)
	}

	err = parser.ParseArgs2(args[1:], false, false)
	if err != nil {
		log.Fatalf("Parse arguments error: %v", err)
	}

	var optionsRef *structarg.BaseOptions

	err = reflectutils.FindAnonymouStructPointer(optStruct, &optionsRef)
	if err != nil {
		log.Fatalf("Find common options fail %s", err)
	}

	if optionsRef.Help {
		fmt.Println(parser.HelpString())
		os.Exit(0)
	}

	if len(optionsRef.Config) == 0 {
		for _, p := range []string{"./etc", "/etc/yunion/notify"} {
			confTmp := path.Join(p, configFileName)
			if _, err := os.Stat(confTmp); err == nil {
				optionsRef.Config = confTmp
				break
			}
		}
	}

	if len(optionsRef.Config) > 0 {
		log.Infof("Use configuration file: %s", optionsRef.Config)
		err = parser.ParseFile(optionsRef.Config)
		if err != nil {
			log.Fatalf("Parse configuration file: %v", err)
		}
	}
}

type SBaseOptions struct {
	SockFileDir   string `help:"socket file directory" default:"/etc/yunion/notify"`
	SenderNum     int    `help:"number of sender" default:"5"`
	TemplateDir   string `help:"template directory"`
	LogLevel      string `help:"log level" default:"info" choices:"debug|info|warn|error"`
	LogFilePrefix string `help:"prefix of log files"`

	structarg.BaseOptions
}
