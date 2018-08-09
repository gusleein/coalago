package coalago

import (
	"crypto/md5"
	"io"
	"strconv"
	"strings"
	"time"
)

type CoAPResource struct {
	Method     CoapMethod
	Path       string
	Handler    CoAPResourceHandler
	MediaTypes []MediaType
	Hash       string // Unique Resource ID
}

type CoAPResourceHandler func(message *CoAPMessage) *CoAPResourceHandlerResult

type CoAPResourceHandlerResult struct {
	Payload   CoAPMessagePayload
	Code      CoapCode
	MediaType MediaType
}

func NewResponse(payload CoAPMessagePayload, code CoapCode) *CoAPResourceHandlerResult {
	return &CoAPResourceHandlerResult{Payload: payload, Code: code, MediaType: -1} // -1 means no value
}

func NewCoAPResource(method CoapMethod, path string, handler CoAPResourceHandler) *CoAPResource {
	path = strings.Trim(path, "/ ")

	h := md5.New()
	io.WriteString(h, strconv.FormatInt(time.Now().UnixNano(), 10))
	hash := h.Sum(nil)

	return &CoAPResource{Method: method, Path: path, Handler: handler, Hash: string(hash)}
}

func (resource *CoAPResource) DoesMatchPath(path string) bool {
	path = strings.Trim(path, "/ ")
	return (resource.Path == path)
}

func (resource *CoAPResource) DoesMatchPathAndMethod(path string, method CoapMethod) bool {
	path = strings.Trim(path, "/ ")
	return (resource.Path == path && resource.Method == method)
}
