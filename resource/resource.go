package resource

import (
	"strings"

	m "github.com/coalalib/coalago/message"
)

type CoAPResource struct {
	Method     m.CoapMethod
	Path       string
	Handler    CoAPResourceHandler
	MediaTypes []m.MediaType
}

type CoAPResourceHandlerResult struct {
	Payload   m.CoAPMessagePayload
	Code      m.CoapCode
	MediaType m.MediaType
}

type CoAPResourceHandler func(message *m.CoAPMessage) *CoAPResourceHandlerResult

func NewResponse(payload m.CoAPMessagePayload, code m.CoapCode) *CoAPResourceHandlerResult {
	return &CoAPResourceHandlerResult{Payload: payload, Code: code, MediaType: -1} // -1 means no value
}

func NewCoAPResource(method m.CoapMethod, path string, handler CoAPResourceHandler) *CoAPResource {
	return &CoAPResource{Method: method, Path: strings.Trim(path, "/ "), Handler: handler}
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
