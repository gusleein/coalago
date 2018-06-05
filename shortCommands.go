package coalago

import (
	"errors"
	"net"
	"net/url"

	m "github.com/coalalib/coalago/message"
)

var (
	ErrUndefinedScheme = errors.New("undefined scheme")
)

type Response struct {
	Body []byte
	Code m.CoapCode
}

func (c *Coala) GET(url string) (*Response, error) {
	message, err := constructMessage(m.GET, url)
	if err != nil {
		return nil, err
	}

	return sendMessage(c, message)
}

func (c *Coala) POST(url string, payload []byte) (*Response, error) {
	message, err := constructMessage(m.POST, url)
	if err != nil {
		return nil, err
	}
	message.Payload = m.NewBytesPayload(payload)

	return sendMessage(c, message)
}

func (c *Coala) DELETE(url string) (*Response, error) {
	message, err := constructMessage(m.DELETE, url)
	if err != nil {
		return nil, err
	}

	return sendMessage(c, message)
}

func sendMessage(c *Coala, message *m.CoAPMessage) (*Response, error) {
	resp, err := c.Send(message, message.Recipient)
	if err != nil {
		return nil, err
	}
	r := new(Response)
	r.Body = resp.Payload.Bytes()
	r.Code = resp.Code

	return r, nil
}

func constructMessage(code m.CoapCode, url string) (*m.CoAPMessage, error) {
	path, scheme, queries, addr, err := parseURI(url)
	if err != nil {
		return nil, err
	}
	message := m.NewCoAPMessage(m.CON, code)
	if scheme == "coap" {
		message.SetSchemeCOAP()
	} else if scheme == "coaps" {
		message.SetSchemeCOAPS()
	} else {
		return nil, ErrUndefinedScheme
	}

	message.SetURIPath(path)

	for k, v := range queries {
		message.SetURIQuery(k, v[0])
	}

	message.Recipient = addr

	return message, nil
}

func parseURI(uri string) (path string, scheme string, queries url.Values, addr net.Addr, err error) {
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
