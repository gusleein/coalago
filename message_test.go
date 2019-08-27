package coalago_test

import (
	"encoding/binary"
	"math/rand"

	. "github.com/onsi/ginkgo/extensions/table"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/coalalib/coalago"
)

var _ = Describe("Message", func() {
	Describe("Serialize message", func() {
		var (
			message  *CoAPMessage
			datagram []byte
			err      error
		)

		BeforeEach(func() {
			message = NewCoAPMessage(CON, GET)
			datagram, err = Serialize(message)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			message = nil
		})

		Context("With correct Message ID", func() {
			It("Should correct serialize message id", func() {
				uint16DatagramSlice := binary.BigEndian.Uint16(datagram[2:4])
				Expect(uint16DatagramSlice).Should(Equal(message.MessageID))
			})
		})

		Context("With correct Version", func() {
			It("Should correct serialize version", func() {
				Expect(datagram[0] >> 6).Should(Equal(uint8(1)))
			})
		})

		Context("With Type", func() {
			DescribeTable("Check each type",
				func(expectedType CoapType) {
					message.Type = expectedType
					datagram, err = Serialize(message)
					Expect(err).NotTo(HaveOccurred())
					Expect(datagram[0] >> 4 & 3).Should(Equal(uint8(expectedType)))
				},
				Entry("CON", CON),
				Entry("NON", NON),
				Entry("ACK", ACK),
				Entry("RST", RST),
			)
		})

		Context("With Token Length", func() {
			DescribeTable("Check each any length",
				func(tokenLength int, isOk bool) {
					token := make([]byte, tokenLength)
					rand.Read(token)
					message.Token = token

					datagram, err = Serialize(message)
					Expect(err == nil).Should(Equal(isOk))
				},
				Entry("Token length is zero", 0, true),
				Entry("Token length is valid", 5, true),
				Entry("Token length is maximum", 8, true),
				Entry("Token length is out of range", 9, false),
			)
		})
	})
})
