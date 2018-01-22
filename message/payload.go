package message

import (
	"bytes"
	"encoding/json"
)

// Represents the payload/content of a CoAP Message
type CoAPMessagePayload interface {
	Bytes() []byte
	Length() int
	String() string
}

/**
 * String plain text Payload
 * The most common
 */

// Instantiates a new message payload of type string
func NewStringPayload(s string) CoAPMessagePayload {
	return &StringCoAPMessagePayload{
		content: s,
	}
}

// Represents a message payload containing string value
type StringCoAPMessagePayload struct {
	content string
}

func (p *StringCoAPMessagePayload) Bytes() []byte {
	return bytes.NewBufferString(p.content).Bytes()
}
func (p *StringCoAPMessagePayload) Length() int {
	return len(p.content)
}
func (p *StringCoAPMessagePayload) String() string {
	return p.content
}

/**
 * Bytes Payload
 */

// Represents a message payload containing an array of bytes
func NewBytesPayload(v []byte) CoAPMessagePayload {
	return &BytesPayload{
		content: v,
	}
}

type BytesPayload struct {
	content []byte
}

func (p *BytesPayload) Bytes() []byte {
	return p.content
}
func (p *BytesPayload) Length() int {
	return len(p.content)
}
func (p *BytesPayload) String() string {
	return string(p.content)
}

/**
 * XML Payload
 * Just a copy of String Payload for now
 */

// Represents a message payload containing XML String
type XMLPayload struct {
	StringCoAPMessagePayload
}

/**
 * Empty Payload
 * Just a stub
 */

func NewEmptyPayload() CoAPMessagePayload {
	return &EmptyPayload{}
}

// Represents an empty message payload
type EmptyPayload struct{}

func (p *EmptyPayload) Bytes() []byte {
	return []byte{}
}
func (p *EmptyPayload) Length() int {
	return 0
}
func (p *EmptyPayload) String() string {
	return ""
}

/**
 * JSON Payload
 */

func NewJSONPayload(obj interface{}) CoAPMessagePayload {
	return &JSONPayload{
		obj: obj,
	}
}

// Represents a message payload containing JSON String
type JSONPayload struct {
	obj interface{}
}

func (p *JSONPayload) Bytes() []byte {
	o, err := json.Marshal(p.obj)
	if err != nil {
		log.Error("JSONPayload: Cannot serialize to JSON:", err)
		return []byte{}
	}
	return o
}
func (p *JSONPayload) Length() int {
	return len(p.Bytes())
}
func (p *JSONPayload) String() string {
	return string(p.Bytes())
}
