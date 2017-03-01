package fpcp

import (
	"bytes"
	"container/list"
	"encoding/json"
	"mime"
	"mime/multipart"
	"net/http"
	"sync"
	"time"

	"github.com/jrivets/gorivets"
	"gopkg.in/gin-gonic/gin.v1"
)

// Http Scene Processor is a struct which allows to implement an HTTP server
// on Scene Processor side
type HttpSceneProcessor struct {
	logger gorivets.Logger
	lock   sync.Mutex
	rl     RespListener
	fps    map[string]*fproc

	// Sets how long to hold get request
	getTO time.Duration
}

type fproc struct {
	lock      sync.Mutex
	cond      sync.Cond
	listening bool
	reqs      *list.List
}

func NewHttpSceneProcessor(logger gorivets.Logger, pushToSec int) *HttpSceneProcessor {
	res := new(HttpSceneProcessor)
	res.fps = make(map[string]*fproc)
	res.logger = logger.WithName("fpcp.http")
	pushToSec = gorivets.Max(1, pushToSec)
	res.getTO = time.Duration(pushToSec) * time.Second
	return res
}

func (sp *HttpSceneProcessor) RespListener(rl RespListener) {
	sp.rl = rl
}

func (sp *HttpSceneProcessor) SendReq(fpId string, req *Req) error {
	fp := sp.getFProc(fpId)
	fp.sendReq(req)
	return nil
}

// Serves POST requests from FP
func (sp *HttpSceneProcessor) POSTHandler(c *gin.Context) {
	fpId := c.Param("fpId")
	if fpId == "" {
		sp.logger.Warn("Got POST request with empty fpId value")
		c.JSON(http.StatusBadRequest, "expecting fpId")
		return
	}

	ct := c.Request.Header.Get("Content-Type")
	_, params, err := mime.ParseMediaType(ct)
	if err != nil {
		sp.logger.Warn("could not read content type err=", err)
		c.JSON(http.StatusBadRequest, err)
		return
	}

	reader := multipart.NewReader(c.Request.Body, params["boundary"])
	form, err := reader.ReadForm(50 * 1024 * 1024)
	if err != nil {
		sp.logger.Warn("Could not read form. err=", err)
		c.JSON(http.StatusBadRequest, err)
		return
	}

	jn, ok := form.Value["resp"]
	if !ok || len(jn) != 1 {
		sp.logger.Warn("Expecting resp JSON value")
		c.JSON(http.StatusBadRequest, "Expecting resp JSON value")
		return
	}
	resp := &Resp{}
	err = json.Unmarshal([]byte(jn[0]), resp)
	if err != nil {
		sp.logger.Warn("Could not unmarshal ", jn[0])
		c.JSON(http.StatusBadRequest, "Could not unmarshal response")
		return
	}

	fh, ok := form.File["image"]
	if ok {
		fl, err := fh[0].Open()
		if err != nil {
			sp.logger.Warn("Could not read image data, err=", err)
			c.JSON(http.StatusBadRequest, "Could not read image data")
			return
		}
		bb := &bytes.Buffer{}
		_, err = io.Copy(bb, fl)
		if resp.Image != nil {
			resp.Image.Data = bb.Bytes()
		}
	}

	if sp.rl != nil {
		sp.rl(fpId, resp)
	} else {
		sp.logger.Warn("No response listener, nobody will be notified")
	}

	c.JSON(http.StatusOK, "")
}

// Serves GET requests from FP. It will return:
// 200 - if request is found
// 204 - if there is no request form the timeout
//
// expects .../:fpId - path request
// accepts ?timeout=1234 - query param (timeout in seconds)
func (sp *HttpSceneProcessor) GETHandler(c *gin.Context) {
	fpId := c.Param("fpId")
	if fpId == "" {
		sp.logger.Warn("Got GET request with empty fpId value")
		c.JSON(http.StatusBadRequest, "expecting fpId")
		return
	}

	to := sp.getTO
	qTO := c.Query("timeout")
	if qTO != "" {
		val, err := strconv.Atoi(value)
		if err != nil || val < 0 {
			sp.logger.Warn("Received timeout in query, but cannot parse it to int timeout=", qTO)
		} else {
			to = time.Second * time.Duration(val)
		}
	}
	fp.logger.Debug("Received GET for fpId=", fpId, " will use timeout=", to/time.Second, " sec.")

	fp := sp.getFProc(fpId)
	req := fp.getReq(to)
	if req == nil {
		fp.logger.Debug("No requests for fpId=", fpId)
		c.JSON(http.StatusNoContent, "")
		return
	}
	fp.logger.Debug("Found request fo fpId=", fpId, ", req=", req)
	c.JSON(http.StatusOK, req)
}

func (sp *HttpSceneProcessor) getFProc(fpId string) *frpoc {
	sp.lock.Lock()
	defer sp.lock.Unlock()

	fp, ok := sp.fps[fpId]
	if !ok {
		sp.logger.Debug("New fproc instance by fpId=", fpId)
		fp = new(fproc)
		fp.cond = sync.NewCond(fp.lock)
		fp.reqs = list.New()
	}
	return fp
}

func (fp *fproc) sendReq(req *Req) {
	fp.lock.Lock()
	defer fp.lock.Unlock()

	fp.reqs.PushBack(req)
	if fp.listening {
		fp.cond.Signal()
	}
}

func (fp *fproc) getReq(to time.Duration) *Req {
	fp.lock.Lock()
	defer fp.lock.Unlock()

	if fp.reqs.Len() <= 0 {
		fp.listening = true
		go fp.notifyInTimeout(to)
		fp.cond.Wait()
		fp.listening = false
	}

	if fp.reqs.Len() > 0 {
		e := fp.reqs.Front()
		return fp.reqs.Remove(e).(*Req)
	}
	return nil
}

func (fp *fproc) notifyInTimeout(to time.Duration) {
	time.Sleep(to)

	fp.lock.Lock()
	defer fp.lock.Unlock()

	if fp.listening {
		fp.cond.Signal()
	}
}
