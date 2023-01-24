package encription

import (
	"net"
	"net/url"

	m "github.com/coalalib/coalago/message"
	"github.com/coalalib/coalago/session"
)

func Encrypt(message *m.CoAPMessage, address net.Addr, aead session.AEAD) error {
	if message.Payload != nil && message.Payload.Length() != 0 {
		var associatedData []byte
		message.Payload = m.NewBytesPayload(aead.Seal(message.Payload.Bytes(), message.MessageID, associatedData))
	}

	err := encryptionOptions(message, address, aead)
	if err != nil {
		return err
	}

	return nil
}

func Decrypt(message *m.CoAPMessage, aead session.AEAD) error {
	if message.Payload != nil && message.Payload.Length() != 0 {
		var associatedData []byte
		newPayload, err := aead.Open(message.Payload.Bytes(), message.MessageID, associatedData)
		if err != nil {
			return err
		}
		message.Payload = m.NewBytesPayload(newPayload)
	}

	return decryptionOptions(message, aead)
}

func encryptionOptions(message *m.CoAPMessage, address net.Addr, aead session.AEAD) error {
	var associatedData []byte

	coapsURI := aead.Seal([]byte(message.GetURI(address.String())), message.MessageID, associatedData)
	message.RemoveOptions(m.OptionURIPath)
	message.RemoveOptions(m.OptionURIQuery)
	message.AddOption(m.OptionСoapsUri, string(coapsURI))

	return nil
}

func decryptionOptions(message *m.CoAPMessage, aead session.AEAD) error {
	coapsURIOption := message.GetOption(m.OptionСoapsUri)
	if coapsURIOption == nil {
		return nil
	}

	var associatedData []byte
	coapsURI, err := aead.Open([]byte(coapsURIOption.StringValue()), message.MessageID, associatedData)
	if err != nil {
		return err
	}

	parsedURL, err := url.Parse(string(coapsURI))
	if err != nil {
		return err
	}
	queries, err := url.ParseQuery(parsedURL.RawQuery)
	if err != nil {
		return err
	}

	message.SetURIPath(parsedURL.Path)

	for k, v := range queries {
		message.SetURIQuery(k, v[0])
	}

	message.RemoveOptions(m.OptionСoapsUri)
	return nil
}
