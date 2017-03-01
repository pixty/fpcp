package fpcp

import (
	"strconv"
)

type (
	Timestamp int64

	RectSize struct {
		W int `json:"w"`
		H int `json:"h"`
	}

	Rect struct {
		L int `json:"l"`
		T int `json:"t"`
		R int `json:"r"`
		B int `json:"b"`
	}

	Image struct {
		Id        string    `json:"id"`
		Size      RectSize  `json:"size"`
		Timestamp Timestamp `json:"timestamp"`
		Data      []byte    `json:"data"`
	}

	Face struct {
		ImgId  string `json:"imgId"`
		Region Rect   `json:"region"`
	}

	Person struct {
		Id          string    `json:"id"`
		FirstSeenAt Timestamp `json:"firstSeenAt"`
		LostAt      Timestamp `json:"lostAt"`
		Faces       []*Face   `json:"faces"`
	}

	Scene struct {
		Timestamp Timestamp `json:"timestamp"`
		Persons   []*Person `json:"persons"`
	}

	Resp struct {
		ReqId  string  `json:"reqId"`
		Error  int     `json:"error"`
		Scene  *Scene  `json:"scene"`
		Image  *Image  `json:"image"`
		Person *Person `json:"person"`
	}

	Req struct {
		ReqId    string `json:"reqId"`
		Scene    bool   `json:"scene"`
		ImgId    string `json:"imgId"`
		PersonId string `json:"personId"`
	}

	RespListener func(fpId string, resp *Resp)
	ReqListener  func(req *Req)

	// The interface is implemented by transport provider from Scene processor side
	SceneProcEnd interface {
		// Upstream coming events
		RespListener(rl RespListener)
		// Downstream sent events
		SendReq(fpId string, req *Req) error
	}

	// This interface is implemented by transport provider for Frame Proc side
	FrameProcEnd interface {
		// Downstream coming events
		ReqListener(rl ReqListener)
		// Upstream sent events
		SendResp(resp *Resp)
	}

	Error int
)

const (
	ERR_NOT_FOUND = 1
	ERR_CLOSED    = 2
)

func CheckError(e error, expErr Error) bool {
	if e == nil {
		return false
	}
	err, ok := e.(Error)
	if !ok {
		return false
	}
	return err == expErr
}

func (e Error) Error() string {
	switch e {
	case ERR_NOT_FOUND:
		return "Not found."
	case ERR_CLOSED:
		return "Already closed"
	}
	return "Unknown. Code=" + strconv.Itoa(int(e))
}
