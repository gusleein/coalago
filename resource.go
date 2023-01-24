package coalago

import (
	"crypto/md5"
	"io"
	"strconv"
	"strings"
	"time"

	m "github.com/coalalib/coalago/message"
)

type CoAPResource struct {
	Method     m.CoapMethod
	Path       string
	Handler    CoAPResourceHandler
	MediaTypes []m.MediaType
	Hash       string // Unique Resource ID
}

type CoAPResourceHandler func(message *m.CoAPMessage) *CoAPResourceHandlerResult

type CoAPResourceHandlerResult struct {
	Payload   m.CoAPMessagePayload
	Code      m.CoapCode
	MediaType m.MediaType
}

func NewResponse(payload m.CoAPMessagePayload, code m.CoapCode) *CoAPResourceHandlerResult {
	return &CoAPResourceHandlerResult{Payload: payload, Code: code, MediaType: -1} // -1 means no value
}

func NewCoAPResource(method m.CoapMethod, path string, handler CoAPResourceHandler) *CoAPResource {
	path = strings.Trim(path, "/ ")

	h := md5.New()
	io.WriteString(h, strconv.FormatInt(time.Now().UnixNano(), 10))
	hash := h.Sum(nil)

	return &CoAPResource{Method: method, Path: path, Handler: handler, Hash: string(hash)}
}

/*
func (resource *CoAPResource) DoesMatchPath(path string) bool {
	path = strings.Trim(path, "/ ")
	return (resource.Path == path)
}

func (resource *CoAPResource) DoesMatchPathAndMethod(path string, method m.CoapMethod) bool {
	path = strings.Trim(path, "/ ")
	return (resource.Path == path && resource.Method == method)
}
*/
