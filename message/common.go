package coalaMsg

import (
	"encoding/binary"
	"errors"
	"math/rand"
	"net"
	"sync/atomic"
	"time"

	cerr "github.com/gusleein/coalago/errors"
	"github.com/gusleein/coalago/util"
)

// GenerateMessageId generate a uint16 Message ID
var currentMessageID int32

func init() {
	rand.Seed(time.Now().UnixNano())
	currentMessageID = int32(rand.Intn(65535))
}

func generateMessageID() uint16 {
	if atomic.LoadInt32(&currentMessageID) < 65535 {
		atomic.AddInt32(&currentMessageID, 1)
	} else {
		atomic.StoreInt32(&currentMessageID, 1)
	}

	return uint16(atomic.LoadInt32(&currentMessageID))
}

func GenerateToken(l int) []byte {
	token := make([]byte, l)
	rand.Read(token)
	return token
}

// type to sort the coap options list (which is mandatory) prior to transmission
type sortOptions []*CoAPMessageOption

func (opts sortOptions) Len() int {
	return len(opts)
}

func (opts sortOptions) Swap(i, j int) {
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

func (opts sortOptions) Less(i, j int) bool {
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
	return 0, errors.New("Invalid Option Delta")
}

// Validates a message object and returns any error upon validation failure
func validateMessage(msg *CoAPMessage) error {
	if msg.Type > 3 {
		return cerr.UnknownMessageType
	}

	if msg.GetTokenLength() > 8 {
		return cerr.InvalidTokenLength
	}

	// Repeated Unrecognized Options
	for _, opt := range msg.Options {
		opts := msg.GetOptions(opt.Code)

		if len(opts) > 1 {
			if !opts[0].IsRepeatableOption() {
				if opts[0].Code&0x01 == 1 {
					return cerr.UnknownCriticalOption
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
func MethodString(c CoapMethod) string {
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
	// case OptionWindowtOffset:
	// 	return "OptionWindowOffset"
	case OptionСoapsUri:
		return "OptionСoapsUri"
	case OptionProxySecurityID:
		return "OptionSecurityID"
	default:
		return "Unknown"
	}
}

func ConstructNextBlock(blockType OptionCode, s *StateSend) (*CoAPMessage, bool) {
	s.Stop = s.Start + s.BlockSize
	if s.Stop > s.Lenght {
		s.Stop = s.Lenght
	}

	blockbyte := s.Payload[s.Start:s.Stop]
	isMore := s.Stop < s.Lenght

	blockMessage := newBlockingMessage(
		s.OrigMessage,
		s.OrigMessage.Recipient,
		blockbyte,
		blockType,
		s.NextNumBlock,
		s.BlockSize,
		s.Windowsize,
		isMore,
	)

	s.NextNumBlock++
	s.Start = s.Stop

	blockMessage.CloneOptions(s.OrigMessage, OptionProxyURI, OptionProxySecurityID)
	blockMessage.ProxyAddr = s.OrigMessage.ProxyAddr

	return blockMessage, !isMore
}

func AckTo(initMessage *CoAPMessage, origMessage *CoAPMessage, code CoapCode) *CoAPMessage {
	result := NewCoAPMessage(ACK, code)
	result.MessageID = origMessage.MessageID
	result.Token = origMessage.Token
	result.CloneOptions(origMessage, OptionURIScheme, OptionSelectiveRepeatWindowSize, OptionBlock1, OptionBlock2, OptionProxySecurityID)
	result.Recipient = origMessage.Sender

	if initMessage != nil {
		result.ProxyAddr = initMessage.ProxyAddr
		result.CloneOptions(initMessage, OptionProxyURI)
	}

	return result
}

func AckToWithWindowOffset(initMessage *CoAPMessage, origMessage *CoAPMessage, code CoapCode, windowSize int, blockNumber int, buf map[int][]byte) *CoAPMessage {
	// var offset = 0
	// for i := blockNumber; i > blockNumber-windowSize; i-- {
	// 	offset++
	// }

	result := NewCoAPMessage(ACK, code)
	result.MessageID = origMessage.MessageID
	result.Token = origMessage.Token
	result.CloneOptions(origMessage, OptionURIScheme, OptionSelectiveRepeatWindowSize, OptionBlock1, OptionBlock2, OptionProxySecurityID)
	result.Recipient = origMessage.Sender

	// if offset > 0 {
	// 	result.AddOption(OptionWindowtOffset, uint16(offset))
	// }

	if initMessage != nil {
		result.ProxyAddr = initMessage.ProxyAddr
		result.CloneOptions(initMessage, OptionProxyURI)
	}

	return result
}

func newBlockingMessage(
	origMessage *CoAPMessage,
	recipient net.Addr,
	frame []byte,
	optionBlock OptionCode,
	blockNum,
	blockSize,
	windowSize int,
	isMore bool,
) *CoAPMessage {
	msg := NewCoAPMessage(CON, origMessage.Code)
	if origMessage.GetScheme() == COAPS_SCHEME {
		msg.SetSchemeCOAPS()
	}

	msg.AddOption(OptionSelectiveRepeatWindowSize, windowSize)
	msg.Payload = NewBytesPayload(frame)
	msg.SetURIPath(origMessage.GetURIPath())
	msg.Token = origMessage.Token

	queries := origMessage.GetOptions(OptionURIQuery)
	msg.AddOptions(queries)

	b := util.NewBlock(isMore, blockNum, blockSize)

	msg.AddOption(optionBlock, b.ToInt())
	msg.Recipient = recipient
	msg.ProxyAddr = origMessage.ProxyAddr

	return msg
}

type StateSend struct {
	Lenght       int
	Offset       int
	Start        int
	Stop         int
	NextNumBlock int
	BlockSize    int
	Windowsize   int
	OrigMessage  *CoAPMessage
	Payload      []byte
}

func NewACKEmptyMessage(message *CoAPMessage, windowSize int) *CoAPMessage {
	emptyAckMessage := NewCoAPMessage(ACK, CoapCodeEmpty)
	emptyAckMessage.Token = message.Token
	emptyAckMessage.MessageID = message.MessageID
	emptyAckMessage.Code = CoapCodeEmpty
	emptyAckMessage.Recipient = message.Recipient
	emptyAckMessage.Payload = NewEmptyPayload()
	// emptyAckMessage.AddOption(OptionSelectiveRepeatWindowSize, windowSize)
	emptyAckMessage.CloneOptions(message, OptionBlock1, OptionBlock2, OptionSelectiveRepeatWindowSize, OptionProxySecurityID)

	return emptyAckMessage
}
