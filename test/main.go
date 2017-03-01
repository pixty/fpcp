package main

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"

	"github.com/jrivets/log4g"
	"gopkg.in/gin-gonic/gin.v1"
)

func main() {
	defer log4g.Shutdown()

	ge := gin.New()
	ge.POST("/stream/:fpId", post)
	go func() {
		time.Sleep(time.Second)
		clientPost()
	}()
	ge.Run("0.0.0.0:5555")

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
	part, err := writer.CreateFormFile("file", filepath.Base("/some-file"))
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
