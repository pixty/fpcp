package fpcp

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

type (
	// Timestamp in milliseconds
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

	// The response listener is on SP side and it is notified every time when
	// the response is received
	RespListener func(fpId string, resp *Resp)

	// The request listener resides on FP side and it is notified every time
	// when request is received
	ReqListener func(req *Req)

	// The interface is implemented by transport provider on SP side
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

	// Scene listener (see SceneProcessor)
	SceneListener func(fpId string, scene *Scene)

	SceneProcessor struct {
		lock   sync.Mutex
		spe    SceneProcEnd
		rmap   map[string]chan *Resp
		reqId  int64
		callTO time.Duration
		sl     SceneListener
	}

	Error int
)

const (
	ERR_NOT_FOUND = 1
	ERR_CLOSED    = 2
	ERR_TIMEOUT   = 3
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
	case ERR_TIMEOUT:
		return "Timeout"
	}
	return "Unknown. Code=" + strconv.Itoa(int(e))
}

func NewSceneProcessor(spe SceneProcEnd, sl SceneListener, callTOSec int) *SceneProcessor {
	sp := new(SceneProcessor)
	sp.spe = spe
	sp.reqId = time.Now().Unix()
	spe.RespListener(sp.onResp)
	sp.rmap = make(map[string]chan *Resp)
	sp.sl = sl
	sp.callTO = time.Duration(callTOSec) * time.Second
	return sp
}

func (sp *SceneProcessor) GetImage(fpId, imgId string) (*Image, error) {
	req, ch := sp.newRequest(false)
	req.ImgId = imgId
	sp.spe.SendReq(fpId, req)
	resp, err := sp.waitResponse(req, ch)
	if err != nil {
		return nil, err
	}
	return resp.Image, nil
}

func (sp *SceneProcessor) GetPerson(fpId, personId string) (*Person, error) {
	req, ch := sp.newRequest(false)
	req.PersonId = personId
	sp.spe.SendReq(fpId, req)
	resp, err := sp.waitResponse(req, ch)
	if err != nil {
		return nil, err
	}
	return resp.Person, nil
}

func (sp *SceneProcessor) RequestScene(fpId string) {
	req, _ := sp.newRequest(true)
	req.Scene = true
	sp.spe.SendReq(fpId, req)
}

func (sp *SceneProcessor) newRequest(async bool) (*Req, chan *Resp) {
	sp.lock.Lock()
	defer sp.lock.Unlock()

	sp.reqId++
	req := new(Req)
	req.ReqId = strconv.FormatInt(sp.reqId, 10)
	var ch chan *Resp
	if !async {
		ch = make(chan *Resp)
		sp.rmap[req.ReqId] = ch
	}
	return req, ch
}

func (sp *SceneProcessor) waitResponse(req *Req, ch chan *Resp) (resp *Resp, err error) {
	select {
	case resp = <-ch:
	case <-time.After(sp.callTO):
		err = Error(ERR_TIMEOUT)
		sp.notify(req.ReqId, nil)
	}

	if resp.Error > 0 {
		err = Error(resp.Error)
		resp = nil
	}

	return resp, err
}

func (sp *SceneProcessor) onResp(fpId string, resp *Resp) {
	if resp.Scene != nil {
		sp.sl(fpId, resp.Scene)
		return
	}

	sp.notify(resp.ReqId, resp)
}

func (sp *SceneProcessor) notify(reqId string, resp *Resp) {
	sp.lock.Lock()
	defer sp.lock.Unlock()

	ch, _ := sp.rmap[reqId]
	if ch != nil {
		ch <- resp
		close(ch)
		delete(sp.rmap, reqId)
	}
}

func (rs RectSize) String() string {
	return "{w=" + strconv.Itoa(rs.W) + ", h=" + strconv.Itoa(rs.H) + "}"
}

func (r Rect) String() string {
	return "{l=" + strconv.Itoa(r.L) + ", t=" + strconv.Itoa(r.T) + ", r=" + strconv.Itoa(r.R) + ", b=" + strconv.Itoa(r.B) + "}"
}

func (i *Image) String() string {
	return fmt.Sprint("{id=", i.Id, ", size=", i.Size, ", ts=", i.Timestamp, ", data=", string(i.Data), "}")
}

func (f *Face) String() string {
	return fmt.Sprint("{imgId=", f.ImgId, ", region=", f.Region, "}")
}

func (p *Person) String() string {
	return fmt.Sprint("{id=", p.Id, ", seenAt=", p.FirstSeenAt, ", lostAt=", p.LostAt, ", faces=", p.Faces, "}")
}

func (s *Scene) String() string {
	return fmt.Sprint("{ts=", s.Timestamp, ", persons=", s.Persons, "}")
}

func (r *Resp) String() string {
	return fmt.Sprint("{reqId=", r.ReqId, ", error=", r.Error, ", image=", r.Image, ", scene=", r.Scene, ", person=", r.Person, "}")
}

func (r *Req) String() string {
	return fmt.Sprint("{reqId=", r.ReqId, ", scene=", r.Scene, ", imageId=", r.ImgId, ", personId=", r.PersonId, "}")
}
