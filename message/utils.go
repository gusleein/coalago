package message

import (
	"encoding/binary"
	"errors"
	"math/rand"
	"time"
)

// GenerateMessageId generate a uint16 Message ID
var currentMessageID uint16

func init() {
	rand.Seed(time.Now().UnixNano())
	currentMessageID = uint16(rand.Intn(65535))
}

//TODO  move to session context
func GenerateMessageID() uint16 {
	if currentMessageID < 65535 {
		currentMessageID++
	} else {
		currentMessageID = 1
	}
	return currentMessageID
}

func RandomMessageID() uint16 {
	return uint16(rand.Intn(65535))
}

// GenerateToken generates a random token by a given length
var genChars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

func GenerateToken(l int) []byte {
	token := make([]rune, l)
	for i := range token {
		token[i] = genChars[rand.Intn(len(genChars))]
	}
	return []byte(string(token))
}

// type to sort the coap options list (which is mandatory) prior to transmission
type SortOptions []*CoAPMessageOption

func (opts SortOptions) Len() int {
	return len(opts)
}

func (opts SortOptions) Swap(i, j int) {
	opts[i], opts[j] = opts[j], opts[i]

	// Check change order of the pathes option.
	if opts[j].Code == OptionURIPath || opts[i].Code == OptionURIPath {
		for index, v := range opts {
			if v.Code == OptionURIPath && index > j && index < i {
				opts[i], opts[index] = opts[index], opts[i]
				opts[j], opts[index] = opts[index], opts[j]
			}
		}
	}
}

func (opts SortOptions) Less(i, j int) bool {
	return opts[i].Code < opts[j].Code
}

func getOptionHeaderValue(optValue int) (int, error) {
	switch true {
	case optValue <= 12:
		return optValue, nil

	case optValue <= 268:
		return 13, nil

	case optValue <= 65804:
		return 14, nil
	}
	log.Error("Invalid Option Delta")
	return 0, errors.New("Invalid Option Delta")
}

// Validates a message object and returns any error upon validation failure
func validateMessage(msg *CoAPMessage) error {
	if msg.Type > 3 {
		return ErrUnknownMessageType
	}

	if msg.GetTokenLength() > 8 {
		return ErrInvalidTokenLength
	}

	// Repeated Unrecognized Options
	for _, opt := range msg.Options {
		opts := msg.GetOptions(opt.Code)

		if len(opts) > 1 {
			if !opts[0].IsRepeatableOption() {
				if opts[0].Code&0x01 == 1 {
					log.Error(opts[0].Code)
					return ErrUnknownCriticalOption
				}
			}
		}
	}

	return nil
}

func valueToBytes(value interface{}) []byte {
	var v uint32

	switch i := value.(type) {
	case string:
		return []byte(i)
	case []byte:
		return i
	case MediaType:
		v = uint32(i)
	case byte:
		v = uint32(i)
	case int:
		v = uint32(i)
	case int32:
		v = uint32(i)
	case uint:
		v = uint32(i)
	case uint32:
		v = i
	default:
		break
	}

	return encodeInt(v)
}

func decodeInt(b []byte) (uint32, error) {
	if len(b) > 4 {
		return 0, errors.New("data outside of type")
	}
	tmp := []byte{0, 0, 0, 0}
	copy(tmp[4-len(b):], b)

	return binary.BigEndian.Uint32(tmp), nil
}

func encodeInt(v uint32) []byte {
	switch {
	case v == 0:
		return nil

	case v < 256:
		return []byte{byte(v)}

	case v < 65536:
		rv := []byte{0, 0}
		binary.BigEndian.PutUint16(rv, uint16(v))
		return rv

	default:
		rv := []byte{0, 0, 0, 0}
		binary.BigEndian.PutUint32(rv, uint32(v))
		return rv
	}
}

// Gets the string representation of a CoAP Method code (e.g. GET, PUT, DELETE etc)
func methodString(c CoapMethod) string {
	switch c {
	case CoapMethodGet:
		return "GET"
	case CoapMethodDelete:
		return "DEL"
	case CoapMethodPost:
		return "POST"
	case CoapMethodPut:
		return "PUT"
	}
	return ""
}

func typeString(c CoapType) string {
	switch c {
	case CON:
		return "CON"
	case NON:
		return "NON"
	case ACK:
		return "ACK"
	case RST:
		return "RST"
	}
	return ""
}

// coapCodeToString returns the string representation of a CoapCode
func coapCodeToString(code CoapCode) string {
	switch code {
	case GET:
		return "GET"
	case POST:
		return "POST"
	case PUT:
		return "PUT"
	case DELETE:
		return "DELETE"
	case CoapCodeEmpty:
		return "0 Empty"
	case CoapCodeCreated:
		return "201 Created"
	case CoapCodeDeleted:
		return "202 Deleted"
	case CoapCodeValid:
		return "203 Valid"
	case CoapCodeChanged:
		return "204 Changed"
	case CoapCodeContent:
		return "205 Content"
	case CoapCodeContinue:
		return "231 Continue"
	case CoapCodeBadRequest:
		return "400 Bad Request"
	case CoapCodeUnauthorized:
		return "401 Unauthorized"
	case CoapCodeBadOption:
		return "402 Bad Option"
	case CoapCodeForbidden:
		return "403 Forbidden"
	case CoapCodeNotFound:
		return "404 Not Found"
	case CoapCodeMethodNotAllowed:
		return "405 Method Not Allowed"
	case CoapCodeNotAcceptable:
		return "406 Not Acceptable"
	case CoapCodePreconditionFailed:
		return "412 Precondition Failed"
	case CoapCodeRequestEntityTooLarge:
		return "413 Request Entity Too Large"
	case CoapCodeUnsupportedContentFormat:
		return "415 Unsupported Content Format"
	case CoapCodeInternalServerError:
		return "500 Internal Server Error"
	case CoapCodeNotImplemented:
		return "501 Not Implemented"
	case CoapCodeBadGateway:
		return "502 Bad Gateway"
	case CoapCodeServiceUnavailable:
		return "503 Service Unavailable"
	case CoapCodeGatewayTimeout:
		return "504 Gateway Timeout"
	case CoapCodeProxyingNotSupported:
		return "505 Proxying Not Supported"
	default:
		return "Unknown"
	}
}

func optionCodeToString(option OptionCode) string {
	switch option {
	case OptionIfMatch:
		return "IfMatch"
	case OptionURIHost:
		return "URIHost"
	case OptionEtag:
		return "Etag"
	case OptionIfNoneMatch:
		return "IfNoneMatch"
	case OptionObserve:
		return "Observe"
	case OptionURIPort:
		return "URIPort"
	case OptionLocationPath:
		return "LocationPath"
	case OptionURIPath:
		return "URIPath"
	case OptionContentFormat:
		return "ContentFormat"
	case OptionMaxAge:
		return "MaxAge"
	case OptionURIQuery:
		return "URIQuery"
	case OptionAccept:
		return "Accept"
	case OptionLocationQuery:
		return "LocationQuery"
	case OptionBlock2:
		return "Block2"
	case OptionBlock1:
		return "Block1"
	case OptionSize2:
		return "Size2"
	case OptionProxyURI:
		return "ProxyURI"
	case OptionProxyScheme:
		return "ProxyScheme"
	case OptionSize1:
		return "Size1"
	case OptionURIScheme:
		return "URIScheme"
	case OptionHandshakeType:
		return "HandshakeType"
	case OptionSessionNotFound:
		return "SessionNotFound"
	case OptionSessionExpired:
		return "SessionExpired"
	case OptionSelectiveRepeatWindowSize:
		return "OptionSelectiveRepeatWindowSize"
	case OptionСoapsUri:
		return "OptionСoapsUri"
	default:
		return "Unknown"
	}
}
