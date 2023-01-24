package newcoala

import (
	"strings"

	m "github.com/coalalib/coalago/message"
)

type CoAPResource struct {
	Method     CoapMethod
	Path       string
	Handler    CoAPResourceHandler
	MediaTypes []MediaType
}

type CoAPResourceHandler func(message *m.CoAPMessage) *CoAPResourceHandlerResult

type CoAPResourceHandlerResult struct {
	Payload   m.CoAPMessagePayload
	Code      CoapCode
	MediaType MediaType
}

func NewResponse(payload m.CoAPMessagePayload, code CoapCode) *CoAPResourceHandlerResult {
	return &CoAPResourceHandlerResult{Payload: payload, Code: code, MediaType: -1} // -1 means no value
}

func NewCoAPResource(method CoapMethod, path string, handler CoAPResourceHandler) *CoAPResource {
	return &CoAPResource{Method: method, Path: strings.Trim(path, "/ "), Handler: handler}
}
