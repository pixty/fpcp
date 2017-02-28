package fpcp

import (
	"golang.org/x/net/context"
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

	// The RequestsListener is implemented by Frame Processor and requested
	// by transport implementation to get the data
	RequestsListener interface {
		GetImage(imgId string) (*Image, error)
		GetPerson(personId string) (*Person, error)
		GetScene() *Scene
	}

	FrameProcessor interface {
		// Returns the Frame Processor identifier
		GetId() string
		GetImage(ctx context.Context, imgId string) (*Image, error)
		GetPerson(ctx context.Context, personId string) (*Person, error)
		GetScene(ctx context.Context) (*Scene, error)
	}

	SceneListener func(fp FrameProcessor, scene *Scene)
)

const (
	ERR_NOT_FOUND = 1
)
