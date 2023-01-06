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

package huawei

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/notify-plugins/pkg/sms/driver"
)

func removeCnCode(num string) string {
	if strings.HasPrefix(num, "+86 ") {
		num = strings.TrimSpace(num[4:])
	}
	return num
}

func sendSms(ctx context.Context, appKey, appSecret, srvUrl string, channel string, dest string, templateId string, params []string, signature string) error {
	dest = removeCnCode(dest)
	log.Debugf("sendSms: appKey=%s appSecret=%s srvUrl=%s, channel=%s dest=%s templateId=%s params=%s signature=%s", appKey, appSecret, srvUrl, channel, dest, templateId, params, signature)
	cli := httputils.GetDefaultClient()
	urlStr := httputils.JoinPath(srvUrl, "/sms/batchSendSms/v1")
	headers := http.Header{}
	headers.Add("Content-Type", "application/x-www-form-urlencoded")
	headers.Add("Authorization", `WSSE realm="SDP",profile="UsernameToken",type="Appkey"`)
	nonce := utils.GenRequestId(8)
	createdAt := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	hash := sha256.New()
	hash.Write([]byte(nonce + createdAt + appSecret))
	digest := base64.StdEncoding.EncodeToString(hash.Sum(nil))
	wsse := fmt.Sprintf(`UsernameToken Username="%s", PasswordDigest="%s", Nonce="%s", Created="%s"`, appKey, digest, nonce, createdAt)
	headers.Add("X-WSSE", wsse)
	body := map[string]string{
		"from":       channel,
		"to":         dest,
		"templateId": templateId,
		"signature":  signature,
	}
	if len(params) > 0 {
		body["templateParas"] = jsonutils.Marshal(params).String()
	}
	bodyStr := jsonutils.Marshal(body).QueryString()
	resp, err := httputils.Request(cli, ctx, httputils.POST, urlStr, headers, strings.NewReader(bodyStr), true)
	if err != nil {
		return errors.Wrap(err, "httputils.Request")
	}
	defer httputils.CloseResponse(resp)

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "ReadAll body")
	}
	respJson, err := jsonutils.Parse(respBytes)
	if err != nil {
		return errors.Wrap(err, "jsonutils.Parse")
	}
	log.Debugf("resp %s", respJson)
	code, _ := respJson.GetString("code")
	switch code {
	case "000000":
		// success
		return nil
	case "E000102", "E000103", "E000106", "E000111", "E000112":
		return driver.ErrAccessKeyIdNotFound
	case "E000101", "E000104", "E000105":
		return driver.ErrSignatureDoesNotMatch
	case "E200029", "E000510":
		return driver.ErrSignnameInvalid
	default:
		msg, _ := respJson.GetString("description")
		return errors.Wrap(errors.ErrClient, msg)
	}
}
