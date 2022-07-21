module yunion.io/x/notify-plugins

go 1.16

require (
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.684
	github.com/golang/protobuf v1.5.2
	github.com/hugozhu/godingtalk v0.0.0-20190801052409-282448228972
	github.com/xen0n/go-workwx v0.1.1
	google.golang.org/grpc v1.38.0
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/mail.v2 v2.3.2
	yunion.io/x/jsonutils v0.0.0-20220106020632-953b71a4c3a8
	yunion.io/x/log v0.0.0-20201210064738-43181789dc74
	yunion.io/x/onecloud v0.0.0-20220409063207-d4ac70645a5d
	yunion.io/x/pkg v0.0.0-20220406030238-39fbc60d5d4e
	yunion.io/x/structarg v0.0.0-20220312084958-9c6c79c7d1c6
)

replace (
	github.com/hugozhu/godingtalk v0.0.0-20190801052409-282448228972 => github.com/rainzm/godingtalk v0.0.0-20200814070325-9ef7f16afffc
	github.com/xen0n/go-workwx v0.1.1 => github.com/rainzm/go-workwx v0.1.2-0.20200810035240-4b03e1755988
	google.golang.org/grpc => google.golang.org/grpc v1.27.1
	gopkg.in/mail.v2 => github.com/yunionio/mail v0.2.0
)
