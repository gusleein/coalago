package coalago

import (
	"net"
	"net/url"

	cerr "github.com/coalalib/coalago/errors"
	m "github.com/coalalib/coalago/message"
)

type Response struct {
	Body          []byte
	Code          m.CoapCode
	PeerPublicKey []byte
}

type Client struct {
	privateKey []byte
}

func NewClient() *Client {
	c := new(Client)
	return c
}

func NewClientWithPrivateKey(pk []byte) *Client {
	c := NewClient()
	c.privateKey = pk
	return c
}

func (c *Client) GET(url string, options ...*m.CoAPMessageOption) (*Response, error) {
	message, err := constructMessage(m.GET, url)
	message.AddOptions(options)

	if err != nil {
		return nil, err
	}
	return clientSendCONMessage(message, c.privateKey, message.Recipient.String())
}

func (c *Client) Send(message *m.CoAPMessage, addr string, options ...*m.CoAPMessageOption) (*Response, error) {
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
	case m.NON, m.ACK:
		return nil, nil
	}
	r := new(Response)
	r.Body = resp.Payload.Bytes()
	r.Code = resp.Code
	r.PeerPublicKey = resp.PeerPublicKey
	return r, nil
}

func (c *Client) POST(data []byte, url string, options ...*m.CoAPMessageOption) (*Response, error) {
	message, err := constructMessage(m.POST, url)
	if err != nil {
		return nil, err
	}
	message.AddOptions(options)

	message.Payload = m.NewBytesPayload(data)
	return clientSendCONMessage(message, c.privateKey, message.Recipient.String())
}

func (c *Client) DELETE(data []byte, url string, options ...*m.CoAPMessageOption) (*Response, error) {
	message, err := constructMessage(m.DELETE, url)
	if err != nil {
		return nil, err
	}
	message.AddOptions(options)

	return clientSendCONMessage(message, c.privateKey, message.Recipient.String())
}

func clientSendCONMessage(message *m.CoAPMessage, privateKey []byte, addr string) (*Response, error) {
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

func clientSendCON(message *m.CoAPMessage, privateKey []byte, addr string) (resp *m.CoAPMessage, err error) {
	conn, err := globalPoolConnections.Dial(addr)
	if err != nil {
		return nil, err
	}

	defer conn.Close()
	sr := newtransport(conn)
	sr.privateKey = privateKey

	return sr.Send(message)
}

func constructMessage(code m.CoapCode, url string) (*m.CoAPMessage, error) {
	path, scheme, queries, addr, err := parseURI(url)
	if err != nil {
		return nil, err
	}

	message := m.NewCoAPMessage(m.CON, code)
	switch scheme {
	case "coap":
		message.SetSchemeCOAP()
	case "coaps":
		message.SetSchemeCOAPS()
	default:
		return nil, cerr.UndefinedScheme
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

func isBigPayload(message *m.CoAPMessage) bool {
	if message.Payload != nil {
		return message.Payload.Length() > MAX_PAYLOAD_SIZE
	}

	return false
}

func Ping(addr string) (isPing bool, err error) {
	msg := m.NewCoAPMessage(m.CON, m.CoapCodeEmpty)
	resp, err := clientSendCON(msg, nil, addr)
	if err != nil {
		return false, err
	}

	return resp.Type == m.RST && resp.Code == m.CoapCodeEmpty && resp.MessageID == msg.MessageID, nil
}
