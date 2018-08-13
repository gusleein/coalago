package coalago

import (
	"errors"
	"net"
	"net/url"
	"time"

	cache "github.com/patrickmn/go-cache"
)

var (
	ErrUndefinedScheme = errors.New("undefined scheme")
	ErrMaxAttempts     = errors.New("max attempts")
	MAX_PAYLOAD_SIZE   = 1024
)

type Response struct {
	Body          []byte
	Code          CoapCode
	PeerPublicKey []byte
}

type Client struct {
	sessions   *cache.Cache
	privateKey []byte
}

func NewClient() *Client {
	c := new(Client)
	c.sessions = cache.New(SESSIONS_POOL_EXPIRATION, time.Second*10)
	return c
}

func NewClientWithPrivateKey(pk []byte) *Client {
	c := NewClient()
	c.privateKey = pk
	return c
}

func (c *Client) GET(url string, options ...*CoAPMessageOption) (*Response, error) {
	message, err := constructMessage(GET, url)
	message.AddOptions(options)

	if err != nil {
		return nil, err
	}
	return clientSendCONMessage(message, c.privateKey, message.Recipient.String())
}

func (c *Client) Send(message *CoAPMessage, addr string, options ...*CoAPMessageOption) (*Response, error) {
	message.AddOptions(options)

	conn, err := globalPoolConnections.Dial(addr)
	if err != nil {
		return nil, err
	}

	defer conn.Close()

	sr := newtransport(conn)
	sr.privateKey = c.privateKey

	resp, err := sr.Send(message)
	if err != nil {
		return nil, err
	}
	switch message.Type {
	case NON, ACK:
		return nil, nil
	}
	r := new(Response)
	r.Body = resp.Payload.Bytes()
	r.Code = resp.Code
	r.PeerPublicKey = resp.PeerPublicKey
	return r, nil
}

func (c *Client) POST(data []byte, url string, options ...*CoAPMessageOption) (*Response, error) {
	message, err := constructMessage(POST, url)
	if err != nil {
		return nil, err
	}
	message.AddOptions(options)

	message.Payload = NewBytesPayload(data)
	return clientSendCONMessage(message, c.privateKey, message.Recipient.String())
}

func (c *Client) DELETE(data []byte, url string, options ...*CoAPMessageOption) (*Response, error) {
	message, err := constructMessage(DELETE, url)
	if err != nil {
		return nil, err
	}
	message.AddOptions(options)

	return clientSendCONMessage(message, c.privateKey, message.Recipient.String())
}

func clientSendCONMessage(message *CoAPMessage, privateKey []byte, addr string) (*Response, error) {
	resp, err := clientSendCON(message, privateKey, addr)
	if err != nil {
		return nil, err
	}
	r := new(Response)
	r.Body = resp.Payload.Bytes()
	r.Code = resp.Code
	r.PeerPublicKey = resp.PeerPublicKey
	return r, nil
}

func clientSendCON(message *CoAPMessage, privateKey []byte, addr string) (resp *CoAPMessage, err error) {
	conn, err := globalPoolConnections.Dial(addr)
	if err != nil {
		return nil, err
	}

	defer conn.Close()
	sr := newtransport(conn)
	sr.privateKey = privateKey

	return sr.Send(message)
}

func constructMessage(code CoapCode, url string) (*CoAPMessage, error) {
	path, scheme, queries, addr, err := parseURI(url)
	if err != nil {
		return nil, err
	}

	message := NewCoAPMessage(CON, code)
	switch scheme {
	case "coap":
		message.SetSchemeCOAP()
	case "coaps":
		message.SetSchemeCOAPS()
	default:
		return nil, ErrUndefinedScheme
	}

	message.SetURIPath(path)

	for k, v := range queries {
		message.SetURIQuery(k, v[0])
	}

	message.Recipient = addr

	return message, nil
}

func parseURI(uri string) (path, scheme string, queries url.Values, addr net.Addr, err error) {
	var u *url.URL
	u, err = url.Parse(uri)
	if err != nil {
		return
	}

	path = u.Path
	scheme = u.Scheme

	queries, err = url.ParseQuery(u.RawQuery)
	if err != nil {
		return
	}

	addr, _ = net.ResolveUDPAddr("udp", u.Host)

	return
}

func isBigPayload(message *CoAPMessage) bool {
	if message.Payload != nil {
		return message.Payload.Length() > MAX_PAYLOAD_SIZE
	}

	return false
}
