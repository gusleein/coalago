package coalago_test

import (
	"encoding/binary"

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
					Expect(datagram[0] >> 4 & 3).To(Equal(uint8(expectedType)))
				},
				Entry("CON", CON),
				Entry("NON", NON),
				Entry("ACK", ACK),
				Entry("RST", RST),
			)
		})

	})
})
