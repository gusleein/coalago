package message

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/coalalib/coalago/blockwise"
	"github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("coala.message")
)

// A Message object represents a CoAP payload
type CoAPMessage struct {
	MessageID uint16
	Type      CoapType
	Code      CoapCode
	Payload   CoAPMessagePayload
	Token     []byte
	Options   []*CoAPMessageOption

	Sender    net.Addr
	Recipient net.Addr

	Attempts int
	LastSent time.Time

	IsProxies bool

	PublicKey []byte

	Callback func(*CoAPMessage, error)
}

func NewCoAPMessage(messageType CoapType, messageCode CoapCode) *CoAPMessage {
	return &CoAPMessage{
		MessageID: GenerateMessageID(),
		Type:      messageType,
		Code:      messageCode,
		Payload:   NewEmptyPayload(),
		Token:     GenerateToken(6),
	}
}

func NewCoAPMessageId(messageType CoapType, messageCode CoapCode, messageID uint16) *CoAPMessage {
	return &CoAPMessage{
		MessageID: messageID,
		Type:      messageType,
		Code:      messageCode,
		Token:     GenerateToken(6),
	}
}

// Converts an array of bytes to a Mesasge object.
// An error is returned if a parsing error occurs
func Deserialize(data []byte) (*CoAPMessage, error) {
	msg := &CoAPMessage{}

	dataLen := len(data)
	if dataLen < 4 {
		return msg, ErrPacketLengthLessThan4
	}

	ver := data[DataHeader] >> 6
	if ver != 1 {
		return nil, ErrInvalidCoapVersion
	}

	msg.Type = CoapType(data[DataHeader] >> 4 & 0x03)
	tokenLength := data[DataHeader] & 0x0f
	msg.Code = CoapCode(data[DataCode])

	msg.MessageID = binary.BigEndian.Uint16(data[DataMsgIDStart:DataMsgIDEnd])

	// Token
	if tokenLength > 0 {
		// msg.Token = make([]byte, tokenLength)
		msg.Token = data[DataTokenStart : DataTokenStart+tokenLength]
		// copy(msg.Token, token)
	}

	/*
	    0   1   2   3   4   5   6   7
	   +---------------+---------------+
	   |               |               |
	   |  Option Delta | Option Length |   1 byte
	   |               |               |
	   +---------------+---------------+
	   \                               \
	   /         Option Delta          /   0-2 bytes
	   \          (extended)           \
	   +-------------------------------+
	   \                               \
	   /         Option Length         /   0-2 bytes
	   \          (extended)           \
	   +-------------------------------+
	   \                               \
	   /                               /
	   \                               \
	   /         Option Value          /   0 or more bytes
	   \                               \
	   /                               /
	   \                               \
	   +-------------------------------+
	*/
	tmp := data[DataTokenStart+msg.GetTokenLength():]

	lastOptionID := uint16(0)
	for len(tmp) > 0 {
		if tmp[0] == PayloadMarker {
			tmp = tmp[1:]
			break
		}

		optionDelta := uint16(tmp[0] >> 4)
		optionLength := uint16(tmp[0] & 0x0f)

		tmp = tmp[1:]
		switch optionDelta {
		case 13:
			optionDeltaExtended := uint16(tmp[0]) + uint16(13)
			optionDelta = optionDeltaExtended
			tmp = tmp[1:]

		case 14:
			optionDeltaExtended := binary.BigEndian.Uint16(tmp[:2])
			optionDelta = optionDeltaExtended + uint16(269)
			tmp = tmp[2:]

		case 15:
			return msg, ErrOptionDeltaUsesValue15
		}

		lastOptionID += optionDelta

		switch optionLength {
		case 13:
			optionLengthExtended := uint16(tmp[0]) + uint16(13)
			optionLength = optionLengthExtended
			tmp = tmp[1:]

		case 14:
			optionLengthExtended := binary.BigEndian.Uint16(tmp[:2])
			optionLength = optionLengthExtended + uint16(269)
			tmp = tmp[2:]

		case 15:
			return msg, ErrOptionLengthUsesValue15
		}

		optCode := OptionCode(lastOptionID)
		if optionLength > 0 {
			optionValue := tmp[:optionLength]

			switch optCode {
			case OptionURIScheme, OptionProxyScheme, OptionURIPort, OptionContentFormat, OptionMaxAge, OptionAccept, OptionSize1,
				OptionSize2, OptionBlock1, OptionBlock2, OptionHandshakeType, OptionObserve,
				OptionSessionNotFound, OptionSessionExpired, OptionSelectiveRepeatWindowSize:

				intVal, err := decodeInt(optionValue)
				if err != nil {
					return nil, err
				}
				msg.Options = append(msg.Options, NewOption(optCode, intVal))

			case OptionURIHost, OptionEtag, OptionLocationPath, OptionURIPath, OptionURIQuery,
				OptionLocationQuery, OptionProxyURI, OptionСoapsUri:
				msg.Options = append(msg.Options, NewOption(optCode, string(optionValue)))
			default:
				if lastOptionID&0x01 == 1 {
					log.Error("Unknown Critical Option id " + strconv.Itoa(int(lastOptionID)))
					return msg, ErrUnknownCriticalOption
				}
				log.Error("Unknown Option id " + strconv.Itoa(int(lastOptionID)))
			}

			tmp = tmp[optionLength:]
		} else {
			msg.Options = append(msg.Options, NewOption(optCode, nil))
		}
	}

	msg.Payload = NewBytesPayload(tmp)

	err := validateMessage(msg)

	return msg, err
}

// Converts a message object to a byte array. Typically done prior to transmission
func Serialize(msg *CoAPMessage) ([]byte, error) {
	if option := msg.GetOption(OptionURIScheme); option != nil {
		if option.Value == nil || option.IntValue() != COAPS_SCHEME {
			msg.AddOption(OptionURIScheme, COAP_SCHEME)
		}
	}

	messageID := []byte{0, 0}
	binary.BigEndian.PutUint16(messageID, msg.MessageID)

	buf := bytes.Buffer{}
	buf.Write([]byte{(1 << 6) | (uint8(msg.Type) << 4) | 0x0f&msg.GetTokenLength()})
	buf.Write([]byte{byte(msg.Code)})
	buf.Write([]byte{messageID[0]})
	buf.Write([]byte{messageID[1]})
	buf.Write(msg.Token)

	// Sort Options
	sort.Sort(SortOptions(msg.Options))

	lastOptionCode := 0
	for _, opt := range msg.Options {
		optCode := int(opt.Code)
		optDelta := optCode - lastOptionCode
		optDeltaValue, _ := getOptionHeaderValue(optDelta)
		byteValue := valueToBytes(opt.Value)
		valueLength := len(byteValue)
		optLength := valueLength
		optLengthValue, _ := getOptionHeaderValue(optLength)

		// Option Header
		buf.Write([]byte{byte(optDeltaValue<<4 | optLengthValue)})

		// Extended Delta & Length
		if optDeltaValue == 13 {
			optDelta -= 13
			buf.Write([]byte{byte(optDelta)})
		} else if optDeltaValue == 14 {
			tmpBuf := new(bytes.Buffer)
			optDelta -= 269
			binary.Write(tmpBuf, binary.BigEndian, uint16(optDelta))
			buf.Write(tmpBuf.Bytes())
		}

		if optLengthValue == 13 {
			optLength -= 13
			buf.Write([]byte{byte(optLength)})
		} else if optLengthValue == 14 {
			tmpBuf := new(bytes.Buffer)
			optLength -= 269
			binary.Write(tmpBuf, binary.BigEndian, uint16(optLength))
			buf.Write(tmpBuf.Bytes())
		}

		// Option Value
		buf.Write(byteValue)
		lastOptionCode = optCode
	}

	if msg.Payload != nil && msg.Payload.Length() > 0 {
		buf.Write([]byte{PayloadMarker})
		buf.Write(msg.Payload.Bytes())
	}

	return buf.Bytes(), nil
}

func (m *CoAPMessage) Clone(includePayload bool) *CoAPMessage {
	cloneMessage := NewCoAPMessageId(m.Type, m.Code, m.MessageID)
	cloneMessage.Token = m.Token
	cloneMessage.Options = m.Options
	if includePayload {
		cloneMessage.Payload = m.Payload
	}
	return cloneMessage
}

func (m *CoAPMessage) GetScheme() int {
	option := m.GetOption(OptionURIScheme)
	if option != nil && option.Value != nil && option.IntValue() == COAPS_SCHEME {
		return COAPS_SCHEME
	}
	return COAP_SCHEME
}

func (m *CoAPMessage) GetSchemeString() string {
	option := m.GetOption(OptionURIScheme)
	if option != nil && option.Value != nil && option.IntValue() == COAPS_SCHEME {
		return "coaps"
	}
	return "coap"
}

func (m *CoAPMessage) GetURI(host string) string {
	result := m.GetSchemeString() + "://" + host + m.GetURIPath()
	query := m.GetURIQueryString()
	if len(query) > 0 {
		result += "?" + query
	}
	return result
}

func (m *CoAPMessage) GetMethod() CoapMethod {
	switch m.Code {
	case GET:
		return CoapMethodGet
	case POST:
		return CoapMethodPost
	case PUT:
		return CoapMethodPut
	case DELETE:
		return CoapMethodDelete
	default:
		return 0
	}
}

func (m *CoAPMessage) GetURIHost() string {
	option := m.GetOption(OptionURIHost)

	if option == nil {
		return "localhost"
	}

	return option.StringValue()
}

func (m *CoAPMessage) GetURIPort() int {
	option := m.GetOption(OptionURIPort)

	if option == nil {
		return 0
	}

	return option.IntValue()
}

func (m *CoAPMessage) GetURIPath() string {
	opts := m.GetOptionsAsString(OptionURIPath)

	return "/" + strings.Join(opts, "/")
}

func (m *CoAPMessage) GetURIQueryString() string {
	options := m.GetOptions(OptionURIQuery)

	var query []string
	for _, v := range options {
		query = append(query, v.StringValue())
	}

	return strings.Join(query, "&")
}

func (m *CoAPMessage) GetURIQueryArray() []string {
	options := m.GetOptions(OptionURIQuery)

	var query []string
	for _, v := range options {
		query = append(query, v.StringValue())
	}

	return query
}

func (m *CoAPMessage) GetURIQuery(q string) string {
	qs := m.GetURIQueryArray()

	for _, v := range qs {
		kv := strings.SplitN(v, "=", 2)
		if len(kv) == 2 {
			if kv[0] == q {
				return kv[1]
			}
		}
	}

	return ""
}

func (m *CoAPMessage) GetCodeString() string {
	codeClass := string(m.Code >> 5)
	codeDetail := string(m.Code & 0x1f)

	return codeClass + "." + codeDetail
}

func (m *CoAPMessage) GetTokenLength() uint8 {
	return uint8(len(m.Token))
}

func (m *CoAPMessage) GetTokenString() string {
	return string(m.Token[:])
}

func (m *CoAPMessage) GetMessageIDString() string {
	return strconv.Itoa(int(m.MessageID))
}

func (m *CoAPMessage) GetPayload() []byte {
	return m.Payload.Bytes()
}

func (m *CoAPMessage) SetProxyURI(uri string) {
	m.AddOption(OptionProxyURI, uri)
}

func (m *CoAPMessage) SetProxyScheme(uri string) {
	m.AddOption(OptionProxyScheme, uri)
}

func (m *CoAPMessage) SetMediaType(mt MediaType) {
	m.AddOption(OptionContentFormat, mt)
}

func (m *CoAPMessage) SetStringPayload(s string) {
	m.Payload = NewStringPayload(s)
}

func (m *CoAPMessage) SetURIPath(fullPath string) {
	pathParts := strings.Split(fullPath, "/")

	for _, path := range pathParts {
		if path != "" {
			m.AddOption(OptionURIPath, path)
		}
	}
}

func (m *CoAPMessage) SetURIQuery(k string, v string) {
	m.AddOption(OptionURIQuery, k+"="+v)
}

func (m *CoAPMessage) SetToken(t string) {
	m.Token = []byte(t)
}

func (message *CoAPMessage) SetSchemeCOAP() {
	message.AddOption(OptionURIScheme, COAP_SCHEME)
}
func (message *CoAPMessage) SetSchemeCOAPS() {
	message.AddOption(OptionURIScheme, COAPS_SCHEME)
}

func (m *CoAPMessage) IsRequest() bool {
	return m.Type == CON
}

func (m *CoAPMessage) ToReadableString() string {
	options := ""
	for _, option := range m.Options {
		options += fmt.Sprintf("%v: '%v' ", optionCodeToString(option.Code), option.Value)
	}

	return fmt.Sprintf(
		"%v\t%v\t%v\t%v\t%v\t[%v]",
		typeString(m.Type),
		coapCodeToString(m.Code),
		m.GetSchemeString(),
		m.GetTokenString(),
		m.MessageID,
		options)
}

func (m *CoAPMessage) GetProxyKeyReceiver() string {
	return m.GetTokenString() + m.Sender.String()
}

func (m *CoAPMessage) GetProxyKeySender(address net.Addr) string {
	return m.GetTokenString() + address.String()
}

func (m *CoAPMessage) GetACKKeyForSend(address net.Addr) string {
	return address.String() + m.GetTokenString() + m.GetMessageIDString()
}
func (m *CoAPMessage) GetACKKeyForReceive() string {
	return m.Sender.String() + m.GetTokenString() + m.GetMessageIDString()
}

func (m *CoAPMessage) IsProxied() bool {
	return m.GetOption(OptionProxyURI) != nil
}

func (m *CoAPMessage) GetBlock1() *blockwise.Block {
	optionBlock1 := m.GetOption(OptionBlock1)
	if optionBlock1 != nil {
		return blockwise.NewBlockFromInt(optionBlock1.IntValue())
	}
	return nil
}

func (m *CoAPMessage) GetBlock2() *blockwise.Block {
	optionBlock2 := m.GetOption(OptionBlock2)
	if optionBlock2 != nil {
		return blockwise.NewBlockFromInt(optionBlock2.IntValue())
	}
	return nil
}

func ParseQuery(query string) (values map[string][]string) {
	values = make(map[string][]string)
	for query != "" {
		params := strings.SplitN(query, "&", 2)

		switch len(params) {
		case 0, 1:
			query = ""
		case 2:
			query = params[1]
		}

		processParams(params[0], values)
	}

	return values
}

func processParams(p string, values map[string][]string) {
	kv := strings.SplitN(p, "=", 2)
	if len(kv) != 2 {
		return
	}

	key := kv[0]
	value := unescapeString(kv[1])

	if values[key] == nil {
		values[key] = []string{}
	}
	values[key] = append(values[key], value)
}

func EscapeString(s string) string {
	newString := ""
	for _, char := range s {
		c := escapeChar(string(char))
		newString = newString + c
	}

	return newString
}

func escapeChar(s string) string {
	switch s {
	case "&":
		return "%26"
	}

	return s
}

func unescapeString(s string) string {
	return strings.Replace(s, "%26", "&", -1)
}
