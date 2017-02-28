package fpcp

import (
	"gopkg.in/gin-gonic/gin.v1"
)

// Http Scene Processor is a struct which allows to implement an HTTP server
// on Scene Processor side
type HttpSceneProcessor struct {
	sl SceneListener
}

func NewHttpSceneProcessor() *HttpSceneProcessor {
	return new(HttpSceneProcessor)
}

func (sp *HttpSceneProcessor) WithListener(sl SceneListener) *HttpSceneProcessor {
	sp.sl = sl
	return sp
}

// Serves POST requests from FP
func (sp *HttpSceneProcessor) POSTHandler(c *gin.Context) {

}

// Serves GET requests from FP
func (sp *HttpSceneProcessor) GETHandler(c *gin.Context) {

}
