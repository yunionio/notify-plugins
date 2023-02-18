module yunion.io/x/notify-plugins

go 1.18

require (
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.684
	github.com/golang/protobuf v1.5.2
	github.com/hugozhu/godingtalk v0.0.0-20190801052409-282448228972
	github.com/xen0n/go-workwx v0.1.1
	google.golang.org/grpc v1.38.0
	gopkg.in/mail.v2 v2.3.1
	yunion.io/x/jsonutils v1.0.1-0.20220819091305-3bab322ab4fd
	yunion.io/x/log v1.0.0
	yunion.io/x/onecloud v0.0.0-20230109063135-d1246adcf9dd
	yunion.io/x/pkg v1.0.1-0.20230102060551-df05ccecb71c
	yunion.io/x/structarg v0.0.0-20220312084958-9c6c79c7d1c6
)

require (
	github.com/cenkalti/backoff/v4 v4.0.0 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mattn/go-colorable v0.1.9 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	github.com/texttheater/golang-levenshtein v0.0.0-20180516184445-d188e65d659e // indirect
	golang.org/x/crypto v0.1.0 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/term v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	google.golang.org/genproto v0.0.0-20210602131652-f16073e35f0c // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	moul.io/http2curl/v2 v2.3.0 // indirect
)

replace (
	github.com/hugozhu/godingtalk v0.0.0-20190801052409-282448228972 => github.com/rainzm/godingtalk v0.0.0-20200814070325-9ef7f16afffc
	github.com/jaypipes/ghw => github.com/zexi/ghw v0.9.1
	github.com/xen0n/go-workwx v0.1.1 => github.com/rainzm/go-workwx v0.1.2-0.20200810035240-4b03e1755988
	google.golang.org/grpc => google.golang.org/grpc v1.27.1
)
