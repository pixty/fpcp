package fpcp

import (
	"fmt"
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
