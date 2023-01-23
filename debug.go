package coalago

/*
  Debug  functions unusable at production!
*/

// //TODO: delete afer test
// func (coala *Coala) initResourceTestsMirror() {
// 	coala.POST("/tests/mirror", func(message   *CoAPMessage) *resource.CoAPResourceHandlerResult {
// 		return resource.NewResponse(message.Payload, CoapCodeContent)
// 	})
// }

// //TODO: delete afer test
// func (coala *Coala) initResourceTestsBlock2() {
// 	coala.GET("/tests/large", func(message   *CoAPMessage) *resource.CoAPResourceHandlerResult {
// 		sizeStr := message.GetURIQuery("size")
// 		size, err := strconv.Atoi(sizeStr)
// 		if err != nil {
// 			size = 1024
// 		}
// 		body := make([]byte, size)
// 		rand.Read(body)

// 		//body := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb") //ccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccdddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd")
// 		return resource.NewResponse( NewBytesPayload(body), CoapCodeContent)
// 	})

// 	coala.POST("/tests/large", func(message   *CoAPMessage) *resource.CoAPResourceHandlerResult {
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

// 		return resource.NewResponse( NewStringPayload(resp), CoapCodeContent)
// 	})
// }
