package message

const PayloadMarker = 0xff

const (
	COAP_SCHEME  = 0
	COAPS_SCHEME = 1
)

type CoapType uint8

const (
	CON CoapType = 0
	NON CoapType = 1
	ACK CoapType = 2
	RST CoapType = 3
)

type CoapMethod uint8

const (
	CoapMethodGet    CoapMethod = 1
	CoapMethodPut    CoapMethod = 2
	CoapMethodPost   CoapMethod = 3
	CoapMethodDelete CoapMethod = 4
)

type CoapCode uint8

const (

	//methods
	GET    CoapCode = 1
	POST   CoapCode = 2
	PUT    CoapCode = 3
	DELETE CoapCode = 4

	// Response
	CoapCodeEmpty    CoapCode = 0
	CoapCodeCreated  CoapCode = 65
	CoapCodeDeleted  CoapCode = 66
	CoapCodeValid    CoapCode = 67
	CoapCodeChanged  CoapCode = 68
	CoapCodeContent  CoapCode = 69
	CoapCodeContinue CoapCode = 95 // (2.31 Continue)

	// Errors
	CoapCodeBadRequest               CoapCode = 128
	CoapCodeUnauthorized             CoapCode = 129
	CoapCodeBadOption                CoapCode = 130
	CoapCodeForbidden                CoapCode = 131
	CoapCodeNotFound                 CoapCode = 132
	CoapCodeMethodNotAllowed         CoapCode = 133
	CoapCodeNotAcceptable            CoapCode = 134
	CoapCodeRequestEntityIncomplete  CoapCode = 136 // (4.08)
	CoapCodeConflict                 CoapCode = 137
	CoapCodePreconditionFailed       CoapCode = 140
	CoapCodeRequestEntityTooLarge    CoapCode = 141
	CoapCodeUnsupportedContentFormat CoapCode = 143
	CoapCodeInternalServerError      CoapCode = 160
	CoapCodeNotImplemented           CoapCode = 161
	CoapCodeBadGateway               CoapCode = 162
	CoapCodeServiceUnavailable       CoapCode = 163
	CoapCodeGatewayTimeout           CoapCode = 164
	CoapCodeProxyingNotSupported     CoapCode = 165
)

func (c *CoapCode) IsRegisteredMethod() bool {
	return (*c > 0 && *c <= 4)
}

func (c *CoapCode) IsCommonError() bool {
	return (*c >= 128 && *c < 160)
}

func (c *CoapCode) IsInternalError() bool {
	return (*c >= 160 && *c <= 165)
}

type MediaType int

const (
	MediaTypeTextPlain                  MediaType = 0
	MediaTypeTextXML                    MediaType = 1
	MediaTypeTextCsv                    MediaType = 2
	MediaTypeTextHTML                   MediaType = 3
	MediaTypeImageGif                   MediaType = 21
	MediaTypeImageJpeg                  MediaType = 22
	MediaTypeImagePng                   MediaType = 23
	MediaTypeImageTiff                  MediaType = 24
	MediaTypeAudioRaw                   MediaType = 25
	MediaTypeVideoRaw                   MediaType = 26
	MediaTypeApplicationLinkFormat      MediaType = 40
	MediaTypeApplicationXML             MediaType = 41
	MediaTypeApplicationOctetStream     MediaType = 42
	MediaTypeApplicationRdfXML          MediaType = 43
	MediaTypeApplicationSoapXML         MediaType = 44
	MediaTypeApplicationAtomXML         MediaType = 45
	MediaTypeApplicationXmppXML         MediaType = 46
	MediaTypeApplicationExi             MediaType = 47
	MediaTypeApplicationFastInfoSet     MediaType = 48
	MediaTypeApplicationSoapFastInfoSet MediaType = 49
	MediaTypeApplicationJSON            MediaType = 50
	MediaTypeApplicationXObitBinary     MediaType = 51
	MediaTypeTextPlainVndOmaLwm2m       MediaType = 1541
	MediaTypeTlvVndOmaLwm2m             MediaType = 1542
	MediaTypeJSONVndOmaLwm2m            MediaType = 1543
	MediaTypeOpaqueVndOmaLwm2m          MediaType = 1544
)

type CoapHandshakeType uint8

const (
	CoapHandshakeTypeClientHello     = 1
	CoapHandshakeTypePeerHello       = 2
	CoapHandshakeTypeClientSignature = 3
	CoapHandshakeTypePeerSignature   = 4
)

type OptionCode int

const (
	OptionIfMatch       OptionCode = 1
	OptionURIHost       OptionCode = 3
	OptionEtag          OptionCode = 4
	OptionIfNoneMatch   OptionCode = 5
	OptionObserve       OptionCode = 6
	OptionURIPort       OptionCode = 7
	OptionLocationPath  OptionCode = 8
	OptionURIPath       OptionCode = 11
	OptionContentFormat OptionCode = 12
	OptionMaxAge        OptionCode = 14
	OptionURIQuery      OptionCode = 15
	OptionAccept        OptionCode = 17
	OptionLocationQuery OptionCode = 20
	OptionBlock2        OptionCode = 23
	OptionBlock1        OptionCode = 27
	OptionSize2         OptionCode = 28
	OptionProxyURI      OptionCode = 35
	OptionProxyScheme   OptionCode = 39
	OptionSize1         OptionCode = 60

	/// URI scheme options specifies scheme to be used for message transmission
	/// See `CoAPMessage.GetScheme()`. Scheme is stored using it's raw value
	OptionURIScheme OptionCode = 2111

	/// Handshake option is used by Coala library to detect handshake CoAP messages
	OptionHandshakeType OptionCode = 3999

	/// Session Not Found option indicates to sender that peer has no active coaps:// session.
	/// Upon receiving the message with this option sender must restart the session
	OptionSessionNotFound OptionCode = 4001

	/// Session expired option indicates that peer's coaps:// session expired
	/// Upon receiving the message with this option sender must restart the session
	OptionSessionExpired OptionCode = 4003

	OptionSelectiveRepeatWindowSize OptionCode = 3001

	OptionСoapsUri = 4005
)

// Fragments/parts of a CoAP Message packet
const (
	DataHeader     = 0
	DataCode       = 1
	DataMsgIDStart = 2
	DataMsgIDEnd   = 4
	DataTokenStart = 4
)
