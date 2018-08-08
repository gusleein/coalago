package coalago

/*
  Debug  functions unusable at production!
*/

// //TODO: delete afer test
// func (coala *Coala) initResourceTestsMirror() {
// 	coala.AddPOSTResource("/tests/mirror", func(message *m.CoAPMessage) *resource.CoAPResourceHandlerResult {
// 		return resource.NewResponse(message.Payload, m.CoapCodeContent)
// 	})
// }

// //TODO: delete afer test
// func (coala *Coala) initResourceTestsBlock2() {
// 	coala.AddGETResource("/tests/large", func(message *m.CoAPMessage) *resource.CoAPResourceHandlerResult {
// 		sizeStr := message.GetURIQuery("size")
// 		size, err := strconv.Atoi(sizeStr)
// 		if err != nil {
// 			size = 1024
// 		}
// 		body := make([]byte, size)
// 		rand.Read(body)

// 		//body := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb") //ccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccdddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd")
// 		return resource.NewResponse(m.NewBytesPayload(body), m.CoapCodeContent)
// 	})

// 	coala.AddPOSTResource("/tests/large", func(message *m.CoAPMessage) *resource.CoAPResourceHandlerResult {
// 		hash := message.GetURIQuery("hash")

// 		var (
// 			md5Hash []byte
// 			resp    string
// 		)
// 		if message.Payload != nil {
// 			h := md5.Sum(message.Payload.Bytes())
// 			md5Hash = h[:]
// 		}

// 		if hex.EncodeToString(md5Hash) == hash {
// 			resp = "SUCCESSFUL"
// 		} else {
// 			resp = "FAILED"
// 		}

// 		return resource.NewResponse(m.NewStringPayload(resp), m.CoapCodeContent)
// 	})
// }
