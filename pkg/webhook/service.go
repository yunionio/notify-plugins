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

package webhook

import "yunion.io/x/notify-plugins/pkg/common"

type SOptions struct {
	common.SBaseOptions
	Insecure bool `help:"insecure for http client"`
	Timeout  int  `help:"timeout for http client" default:"60"`
}

func StartService() {
	var config SOptions
	common.StartServiceForRobot(&config, NewSender, nil, "webhook", "")
}
