package cerr

import "errors"

var (
	MaxAttempts                   = errors.New("max attempts")
	SessionNotFound               = errors.New("session not found")
	ClientSessionNotFound         = errors.New("client session not found")
	SessionExpired                = errors.New("session expired")
	ClientSessionExpired          = errors.New("client session expired")
	Handshake                     = errors.New("error handshake")
	PacketLengthLessThan4         = errors.New("Packet length less than 4 bytes")
	InvalidCoapVersion            = errors.New("Invalid CoAP version. Should be 1.")
	OptionLengthUsesValue15       = errors.New("Message format error. Option length has reserved value of 15")
	OptionDeltaUsesValue15        = errors.New("Message format error. Option delta has reserved value of 15")
	UnknownMessageType            = errors.New("Unknown message type")
	InvalidTokenLength            = errors.New("Invalid Token Length ( > 8)")
	UnknownCriticalOption         = errors.New("Unknown critical option encountered")
	UnsupportedMethod             = errors.New("Unsupported Method")
	NoMatchingRoute               = errors.New("No matching route found")
	UnsupportedContentFormat      = errors.New("Unsupported Content-Format")
	NoMatchingMethod              = errors.New("No matching method")
	NilMessage                    = errors.New("Message is nil")
	RepeatedMessage               = errors.New("Repeated message")
	NilConn                       = errors.New("Connection object is nil")
	NilAddr                       = errors.New("Address cannot be nil")
	OptionLenghtOutOfRangePackets = errors.New("Option lenght out of range packet")
	UndefinedScheme               = errors.New("Undefined scheme")
	UnsupportedType               = errors.New("Unsuported type")
	ERR_KEYS_NOT_MATCH            = "Expected and current public keys do not match"
)
