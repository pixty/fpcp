package main

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	//	"strconv"
	//	"time"

	"github.com/pixty/fpcp"

	"github.com/jrivets/log4g"
	"gopkg.in/gin-gonic/gin.v1"
)

func main() {
	defer log4g.Shutdown()

	ge := gin.New()
	//	ge.POST("/stream/:fpId", post)
	//	go func() {
	//		time.Sleep(time.Second)
	//		clientPost()
	//	}()

	log4g.SetLogLevel("fpcp", log4g.TRACE)

	sp := fpcp.NewHttpSceneProcessor(log4g.GetLogger(""), 10)
	sp.RespListener(onResp)
	fp := fpcp.NewHttpFrameProcessor(log4g.GetLogger(""), "jopa", "http://127.0.0.1:5555/fpcp/", 30, 5)
	fp.ReqListener(onReq)

	ge.POST("/fpcp/:fpId", sp.POSTHandler)
	ge.GET("/fpcp/:fpId", sp.GETHandler)
	//	go func() {
	//		time.Sleep(time.Second)
	//		log := log4g.GetLogger("test")

	//		image := &fpcp.Image{}
	//		image.Id = "ImageTestId"
	//		image.Size = fpcp.RectSize{100, 200}
	//		image.Data = []byte("tested data")

	//		resp := &fpcp.Resp{}
	//		resp.ReqId = "ImageTestId"
	//		resp.Error = 1234
	//		resp.Image = image

	//		log.Info("Sending response ", resp)
	//		err := fp.SendResp(resp)
	//		if err != nil {
	//			log.Error("Got error while sending response err=", err)
	//		}

	//		req := &fpcp.Req{}
	//		req.ImgId = "1234"
	//		req.ReqId = "1234123"
	//		req.Scene = true
	//		err = sp.SendReq("1234", req)
	//		if err != nil {
	//			log.Error("Got error while sending request err=", err)
	//		}

	//		time.Sleep(10 * time.Second)
	//		fp.Close()

	//	}()
	ge.Run("0.0.0.0:5555")

}

func onResp(fpId string, resp *fpcp.Resp) {
	log := log4g.GetLogger("onResponse")
	log.Info("GOT ", resp)

	if resp.Scene != nil && resp.Image != nil && resp.Scene.ImageId == resp.Image.Id {
		log.Info("Got scene with an Image")
		fn := "/Users/dmitry/frame_processor/image" + resp.Image.Id + ".png"
		f, err := os.Create(fn)
		if err == nil {
			n, err := f.Write(resp.Image.Data)
			log.Info(n, " bytes written into ", fn, ", err=", err)
		} else {
			log.Error("Could not open file err=", err)
		}
	}
}

func onReq(req *fpcp.Req) {
	log4g.GetLogger("onRequest").Info("GOT ", req)
}

func post(c *gin.Context) {
	log := log4g.GetLogger("test")
	fpId := c.Param("fpId")

	log.Info("received post fpId=", fpId)

	ct := c.Request.Header.Get("Content-Type")
	_, params, err := mime.ParseMediaType(ct)
	if err != nil {
		log.Error("could not read content type err=", err)
		return
	}
	reader := multipart.NewReader(c.Request.Body, params["boundary"])
	form, err := reader.ReadForm(1000000)
	if err != nil {
		log.Error("Could not read form err=", err)
	}
	for k, v := range form.Value {
		log.Info("read k=", k, " v=", v)
	}
	for fn, file := range form.File {
		log.Info("file=", fn, " len=", len(file))
		if len(file) > 0 {
			fl, err := file[0].Open()
			if err != nil {
				log.Error("could not read first file ")
				continue
			}
			bb := &bytes.Buffer{}
			_, err = io.Copy(bb, fl)
			if err != nil {
				log.Error("Cannot copy data from file")
			} else {
				log.Info("Read from file=", string(bb.Bytes()))
			}
		}
	}

}

func clientPost() error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("jopa", filepath.Base("ttt"))
	if err != nil {
		return err
	}
	bb := bytes.NewBuffer([]byte("the file context2345"))
	_, err = io.Copy(part, bb)

	params := map[string]string{
		"title":       "My Document",
		"author":      "Matt Aimonetti",
		"description": "A document with all the Go programming language secrets",
	}

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "http://localhost:5555/stream/jopa", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	_, err = client.Do(req)
	if err != nil {
		fmt.Println("Vryv err=", err)
	}
	return nil
}
