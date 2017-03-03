package fpcp

import (
	"bytes"
	"container/list"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/jrivets/gorivets"
	"github.com/mohae/deepcopy"
	"golang.org/x/net/context"
	"gopkg.in/gin-gonic/gin.v1"
)

// Http Scene Processor is a struct which allows to implement an HTTP server
// on Scene Processor side
type HttpSceneProcessor struct {
	logger gorivets.Logger
	lock   sync.Mutex
	rl     RespListener
	fps    map[string]*fproc
	ctx    context.Context
	cancel context.CancelFunc

	// Sets how long to hold get request
	getTO time.Duration
}

type fproc struct {
	logger    gorivets.Logger
	lock      sync.Mutex
	cond      *sync.Cond
	listening bool
	reqs      *list.List
	lastTouch time.Time
}

type HttpFrameProcessor struct {
	id        string
	logger    gorivets.Logger
	lock      sync.Mutex
	rl        ReqListener
	url       string
	client    *http.Client
	clntTO    time.Duration
	pollTOSec int
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewHttpSceneProcessor(logger gorivets.Logger, pushToSec int) *HttpSceneProcessor {
	res := new(HttpSceneProcessor)
	res.fps = make(map[string]*fproc)
	res.logger = logger.WithName("fpcp.sp.http").(gorivets.Logger)
	pushToSec = gorivets.Max(1, pushToSec)
	res.getTO = time.Duration(pushToSec) * time.Second
	res.ctx, res.cancel = context.WithCancel(context.Background())
	go res.sweepFPs()
	return res
}

func NewHttpFrameProcessor(logger gorivets.Logger, id, url string, clientToSec, pollTOSec int) *HttpFrameProcessor {
	res := new(HttpFrameProcessor)
	res.id = id
	res.url = url
	res.logger = logger.WithName("fpcp.fp.http").(gorivets.Logger)
	clientToSec = gorivets.Max(1, clientToSec)
	res.clntTO = time.Duration(clientToSec) * time.Second
	res.pollTOSec = pollTOSec
	res.ctx, res.cancel = context.WithCancel(context.Background())
	go res.longPolling()
	return res
}

func (sp *HttpSceneProcessor) RespListener(rl RespListener) {
	sp.rl = rl
}

func (sp *HttpSceneProcessor) SendReq(fpId string, req *Req) error {
	sp.logger.Debug("Sending request to fpId=", fpId, ", req=", req)
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
	fp := sp.getFProc(fpId)
	fp.lastTouch = time.Now()

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
		val, err := strconv.Atoi(qTO)
		if err != nil || val < 0 {
			sp.logger.Warn("Received timeout in query, but cannot parse it to int timeout=", qTO)
		} else {
			to = time.Second * time.Duration(val)
		}
	}
	sp.logger.Debug("Received GET for fpId=", fpId, " will use timeout=", int(to/time.Second), " sec.")

	fp := sp.getFProc(fpId)
	req := fp.getReq(to)
	if req == nil {
		sp.logger.Debug("No requests for fpId=", fpId)
		c.Status(http.StatusNoContent)
		return
	}
	sp.logger.Debug("Found request fo fpId=", fpId, ", req=", req)
	c.JSON(http.StatusOK, req)
}

func (sp *HttpSceneProcessor) getFProc(fpId string) *fproc {
	sp.lock.Lock()
	defer sp.lock.Unlock()

	fp, ok := sp.fps[fpId]
	if !ok {
		sp.logger.Debug("New fproc instance by fpId=", fpId)
		fp = new(fproc)
		fp.cond = sync.NewCond(&fp.lock)
		fp.reqs = list.New()
		fp.lastTouch = time.Now()
		fp.logger = sp.logger.WithId("fpId=" + fpId).(gorivets.Logger)
		sp.fps[fpId] = fp
	}
	return fp
}

func (sp *HttpSceneProcessor) Close() {
	sp.cancel()
}

func (sp *HttpSceneProcessor) sweepFPs() {
	sp.logger.Info("sweeping FPs started")
	defer sp.logger.Info("sweeping FPs stopped")
	for {
		select {
		case <-sp.ctx.Done():
			return
		case <-time.After(sp.getTO):
		}

		now := time.Now()
		to := sp.getTO * time.Duration(3)
		sp.lock.Lock()
		for fpId, fp := range sp.fps {
			if now.Sub(fp.lastTouch) > to {
				sp.logger.Info("Sweeping fpId=", fpId)
				delete(sp.fps, fpId)
			}
		}
		sp.lock.Unlock()
	}
}

func (fp *fproc) sendReq(req *Req) {
	fp.lock.Lock()
	defer fp.lock.Unlock()

	fp.logger.Debug("Pushing request to list, signaling waiter=", fp.listening)
	fp.reqs.PushBack(req)
	if fp.listening {
		fp.cond.Signal()
	}
}

func (fp *fproc) getReq(to time.Duration) *Req {
	fp.lock.Lock()
	defer fp.lock.Unlock()

	fp.lastTouch = time.Now()
	if fp.reqs.Len() <= 0 {
		fp.listening = true
		fp.logger.Debug("Waiting for new request...")
		go fp.notifyInTimeout(to)
		fp.cond.Wait()
		fp.listening = false
		fp.logger.Debug("Done with waiting.")
	}

	if fp.reqs.Len() > 0 {
		e := fp.reqs.Front()
		req := fp.reqs.Remove(e).(*Req)
		fp.logger.Debug("Getting new request from list req=", req)
		return req
	}
	return nil
}

func (fp *fproc) notifyInTimeout(to time.Duration) {
	fp.logger.Debug("notifyInTimeout to=", to)
	defer fp.logger.Debug("notifyInTimeout done.")
	time.Sleep(to)

	fp.lock.Lock()
	defer fp.lock.Unlock()

	if fp.listening {
		fp.cond.Signal()
	}
}

func (hfp *HttpFrameProcessor) ReqListener(rl ReqListener) {
	hfp.rl = rl
}

func (hfp *HttpFrameProcessor) SendResp(resp *Resp) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if resp.Image != nil && resp.Image.Data != nil {
		hfp.logger.Debug("Found not empty image, adding its data...")
		part, err := writer.CreateFormFile("image", filepath.Base("image"))
		if err != nil {
			hfp.logger.Warn("Could not Create form, err=", err)
			return err
		}

		_, err = io.Copy(part, bytes.NewBuffer(resp.Image.Data))
		if err != nil {
			hfp.logger.Warn("Could not copy image data, err=", err)
			return err
		}
		resp = deepcopy.Copy(resp).(*Resp)
		resp.Image.Data = nil
	}

	respJson, err := json.Marshal(resp)
	if err != nil {
		hfp.logger.Warn("Could not marshal response, err=", err)
		return err
	}

	err = writer.WriteField("resp", string(respJson))
	if err != nil {
		hfp.logger.Warn("Could not write response JSON=", respJson, ". err=", err)
		return err
	}

	err = writer.Close()
	if err != nil {
		hfp.logger.Warn("Cannot close writer, err=", err)
		return err
	}

	url := hfp.url + hfp.id
	req, err := http.NewRequest("POST", url, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	c := hfp.getClient()
	_, err = c.Do(req)
	if err != nil {
		hfp.logger.Warn("Cannot send response to url=", url, ", err=", err)
		return err
	}
	return nil
}

func (hfp *HttpFrameProcessor) longPolling() {
	hfp.logger.Info("Start polling process")
	defer hfp.logger.Info("Exit polling process")

	url := hfp.url + hfp.id
	if hfp.pollTOSec > 0 {
		url = url + "?timeout=" + strconv.Itoa(hfp.pollTOSec)
	}

	var wait bool
	for hfp.ctx.Err() == nil {
		if wait {
			hfp.logger.Debug("There was error in previous send, sleep for a second")
			time.Sleep(time.Second)
		}
		wait = false

		c := hfp.getClient()
		hfp.logger.Debug("Sending long poll GET to ", url)

		resp, err := c.Get(url)
		if err != nil {
			hfp.logger.Error("Got error for GET to ", url, ", err=", err)
			hfp.client = nil
			wait = true
			continue
		}

		if resp.StatusCode == http.StatusOK {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				hfp.logger.Warn("Got a body, but could not read it. err=", err)
				wait = true
				continue
			}

			req := &Req{}
			err = json.Unmarshal(body, req)
			if err != nil {
				hfp.logger.Warn("Could not unmarshal body=", body, " to Resp. err=", err)
				wait = true
				continue
			}

			hfp.logger.Debug("Got push notification req=", req)
			hfp.rl(req)
		}
	}
}

func (hfp *HttpFrameProcessor) Close() {
	hfp.cancel()
}

func (hfp *HttpFrameProcessor) getClient() *http.Client {
	c := hfp.client
	if c != nil {
		return c
	}

	hfp.lock.Lock()
	defer hfp.lock.Unlock()

	hfp.client = &http.Client{Timeout: hfp.clntTO}

	return hfp.client
}
