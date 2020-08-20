package godingtalk

import "strings"

const (
	topAPIMsgAsyncSendv2 = "topapi/message/corpconversation/asyncsend_v2"
	topAPIMsgGetprogress = "topapi/message/corpconversation/getsendprogress"
	topAPIMsgGetResult   = "topapi/message/corpconversation/getsendresult"
)

type topAPIMsgSendv2Resp struct {
	OAPIResponse
	TaskID int `json:"task_id"`
}

func (c *DingTalkClient) TopAPIMsgSendv2(userList []string, msg map[string]interface{}) (int, error) {
	var resp topAPIMsgSendv2Resp
	request := map[string]interface{}{
		"agent_id":    c.AgentID,
		"userid_list": strings.Join(userList, ","),
		"msg":         msg,
	}
	c.RefreshAccessToken()
	err := c.httpRPC(topAPIMsgAsyncSendv2, nil, request, &resp)
	if err != nil {
		return 0, err
	}
	return resp.TaskID, nil
}

type TopAPIMsgSendProgress struct {
	Percent int `json:"progress_in_percent"`
	Status  int `json:"status"`
}

type topAPIMsgGetProgressResp struct {
	OAPIResponse
	Progress TopAPIMsgSendProgress `json:"progress"`
}

func (c *DingTalkClient) TopAPIMsgGetSendProgressv2(taskID int) (TopAPIMsgSendProgress, error) {
	var resp topAPIMsgGetProgressResp
	request := map[string]interface{}{
		"agent_id": c.AgentID,
		"task_id":  taskID,
	}
	c.RefreshAccessToken()
	err := c.httpRPC(topAPIMsgGetprogress, nil, request, &resp)
	if err != nil {
		return TopAPIMsgSendProgress{}, err
	}
	return resp.Progress, nil
}

type TopAPIMsgSendResult struct {
	InvalidUserIDList   []string `json:"invalid_user_id_list"`
	ForbiddenUserIDList []string `json:"forbidden_user_id_list"`
	FaildedUserIDList   []string `json:"failed_user_id_list"`
	ReadUserIDLIst      []string `json:"read_user_id_list"`
	UnreadUserIDList    []string `json:"unread_user_id_list"`
	InvalidDeptIDList   []int    `json:"invalid_dept_id_list"`
}

type topAPIMsgGetSendResultResp struct {
	OAPIResponse
	SendResult TopAPIMsgSendResult `json:"send_result"`
}

func (c *DingTalkClient) TopAPIMsgGetSendResultv2(taskID int) (TopAPIMsgSendResult, error) {
	var resp topAPIMsgGetSendResultResp
	request := map[string]interface{}{
		"agent_id": c.AgentID,
		"task_id":  taskID,
	}
	c.RefreshAccessToken()
	err := c.httpRPC(topAPIMsgGetResult, nil, request, &resp)
	if err != nil {
		return TopAPIMsgSendResult{}, err
	}
	return resp.SendResult, nil
}
