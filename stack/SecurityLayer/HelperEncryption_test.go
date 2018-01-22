package SecurityLayer

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/coalalib/coalago/crypto"
	m "github.com/coalalib/coalago/message"
)

func TestEncrypt(t *testing.T) {
	keyAlice := make([]byte, 16)
	keyBob := make([]byte, 16)
	ivAlice := make([]byte, 4)
	ivBob := make([]byte, 4)
	rand.Reader.Read(keyAlice)
	rand.Reader.Read(keyBob)
	rand.Reader.Read(ivAlice)
	rand.Reader.Read(ivBob)

	alice, err := crypto.NewAEAD(keyBob, keyAlice, ivBob, ivAlice)
	if err != nil {
		t.Error(err)
	}

	message := m.NewCoAPMessage(m.ACK, m.CoapCodeContent)

	message.Payload = m.NewBytesPayload(make([]byte, 512))
	Encrypt(message, alice)
	fmt.Println(message.Payload.Length())

}

//func BenchmarkEncrypt10(b *testing.B) { benchmarkEncryption(1024*1, b) }

// func BenchmarkEncrypt20(b *testing.B)  { benchmarkEncryption(1024*2, b) }
// func BenchmarkEncrypt30(b *testing.B)  { benchmarkEncryption(1024*3, b) }
// func BenchmarkEncrypt40(b *testing.B)  { benchmarkEncryption(1024*4, b) }
// func BenchmarkEncrypt50(b *testing.B)  { benchmarkEncryption(1024*5, b) }
// func BenchmarkEncrypt60(b *testing.B)  { benchmarkEncryption(1024*6, b) }
// func BenchmarkEncrypt70(b *testing.B)  { benchmarkEncryption(1024*7, b) }
// func BenchmarkEncrypt80(b *testing.B)  { benchmarkEncryption(1024*8, b) }
// func BenchmarkEncrypt90(b *testing.B)  { benchmarkEncryption(1024*9, b) }
// func BenchmarkEncrypt100(b *testing.B) { benchmarkEncryption(1024*10, b) }
// func BenchmarkEncrypt110(b *testing.B) { benchmarkEncryption(1024*11, b) }
// func BenchmarkEncrypt120(b *testing.B) { benchmarkEncryption(1024*12, b) }
// func BenchmarkEncrypt130(b *testing.B) { benchmarkEncryption(1024*13, b) }
// func BenchmarkEncrypt140(b *testing.B) { benchmarkEncryption(1024*14, b) }
// func BenchmarkEncrypt150(b *testing.B) { benchmarkEncryption(1024*15, b) }
// func BenchmarkEncrypt160(b *testing.B) { benchmarkEncryption(1024*16, b) }
// func BenchmarkEncrypt170(b *testing.B) { benchmarkEncryption(1024*17, b) }
// func BenchmarkEncrypt180(b *testing.B) { benchmarkEncryption(1024*18, b) }
// func BenchmarkEncrypt190(b *testing.B) { benchmarkEncryption(1024*19, b) }
// func BenchmarkEncrypt200(b *testing.B) { benchmarkEncryption(1024*20, b) }
// func BenchmarkEncrypt210(b *testing.B) { benchmarkEncryption(1024*21, b) }
// func BenchmarkEncrypt220(b *testing.B) { benchmarkEncryption(1024*22, b) }
// func BenchmarkEncrypt230(b *testing.B) { benchmarkEncryption(1024*23, b) }
// func BenchmarkEncrypt240(b *testing.B) { benchmarkEncryption(1024*24, b) }
// func BenchmarkEncrypt250(b *testing.B) { benchmarkEncryption(1024*25, b) }
// func BenchmarkEncrypt260(b *testing.B) { benchmarkEncryption(1024*26, b) }
// func BenchmarkEncrypt270(b *testing.B) { benchmarkEncryption(1024*27, b) }
// func BenchmarkEncrypt280(b *testing.B) { benchmarkEncryption(1024*28, b) }
// func BenchmarkEncrypt290(b *testing.B) { benchmarkEncryption(1024*29, b) }
// func BenchmarkEncrypt300(b *testing.B) { benchmarkEncryption(1024*30, b) }

// func BenchmarkEncrypt310(b *testing.B) { benchmarkEncryption(1024*31, b) }
// func BenchmarkEncrypt320(b *testing.B) { benchmarkEncryption(1024*32, b) }
// func BenchmarkEncrypt330(b *testing.B) { benchmarkEncryption(1024*33, b) }
// func BenchmarkEncrypt340(b *testing.B) { benchmarkEncryption(1024*34, b) }
// func BenchmarkEncrypt350(b *testing.B) { benchmarkEncryption(1024*35, b) }
// func BenchmarkEncrypt360(b *testing.B) { benchmarkEncryption(1024*36, b) }
// func BenchmarkEncrypt370(b *testing.B) { benchmarkEncryption(1024*37, b) }
// func BenchmarkEncrypt380(b *testing.B) { benchmarkEncryption(1024*38, b) }
// func BenchmarkEncrypt390(b *testing.B) { benchmarkEncryption(1024*39, b) }

// func BenchmarkEncrypt400(b *testing.B) { benchmarkEncryption(1024*40, b) }
// func BenchmarkEncrypt410(b *testing.B) { benchmarkEncryption(1024*41, b) }
// func BenchmarkEncrypt420(b *testing.B) { benchmarkEncryption(1024*42, b) }
// func BenchmarkEncrypt430(b *testing.B) { benchmarkEncryption(1024*43, b) }
// func BenchmarkEncrypt440(b *testing.B) { benchmarkEncryption(1024*44, b) }
// func BenchmarkEncrypt450(b *testing.B) { benchmarkEncryption(1024*45, b) }
// func BenchmarkEncrypt460(b *testing.B) { benchmarkEncryption(1024*46, b) }
// func BenchmarkEncrypt470(b *testing.B) { benchmarkEncryption(1024*47, b) }
// func BenchmarkEncrypt480(b *testing.B) { benchmarkEncryption(1024*48, b) }
// func BenchmarkEncrypt490(b *testing.B) { benchmarkEncryption(1024*49, b) }
// func BenchmarkEncrypt500(b *testing.B) { benchmarkEncryption(1024*50, b) }
// func BenchmarkEncrypt510(b *testing.B) { benchmarkEncryption(1024*51, b) }
// func BenchmarkEncrypt520(b *testing.B) { benchmarkEncryption(1024*52, b) }
// func BenchmarkEncrypt530(b *testing.B) { benchmarkEncryption(1024*53, b) }
// func BenchmarkEncrypt540(b *testing.B) { benchmarkEncryption(1024*54, b) }
// func BenchmarkEncrypt550(b *testing.B) { benchmarkEncryption(1024*55, b) }
// func BenchmarkEncrypt560(b *testing.B) { benchmarkEncryption(1024*56, b) }
// func BenchmarkEncrypt570(b *testing.B) { benchmarkEncryption(1024*57, b) }
// func BenchmarkEncrypt580(b *testing.B) { benchmarkEncryption(1024*58, b) }
// func BenchmarkEncrypt590(b *testing.B) { benchmarkEncryption(1024*59, b) }
// func BenchmarkEncrypt600(b *testing.B) { benchmarkEncryption(1024*60, b) }

// func BenchmarkEncrypt610(b *testing.B) { benchmarkEncryption(1024*61, b) }
// func BenchmarkEncrypt620(b *testing.B) { benchmarkEncryption(1024*62, b) }
// func BenchmarkEncrypt630(b *testing.B) { benchmarkEncryption(1024*63, b) }
// func BenchmarkEncrypt640(b *testing.B) { benchmarkEncryption(1024*64, b) }
// func BenchmarkEncrypt650(b *testing.B) { benchmarkEncryption(1024*65, b) }
// func BenchmarkEncrypt660(b *testing.B) { benchmarkEncryption(1024*66, b) }
// func BenchmarkEncrypt670(b *testing.B) { benchmarkEncryption(1024*67, b) }
// func BenchmarkEncrypt680(b *testing.B) { benchmarkEncryption(1024*68, b) }
// func BenchmarkEncrypt690(b *testing.B) { benchmarkEncryption(1024*69, b) }
// func BenchmarkEncrypt700(b *testing.B) { benchmarkEncryption(1024*70, b) }
// func BenchmarkEncrypt710(b *testing.B) { benchmarkEncryption(1024*71, b) }
// func BenchmarkEncrypt720(b *testing.B) { benchmarkEncryption(1024*72, b) }
// func BenchmarkEncrypt730(b *testing.B) { benchmarkEncryption(1024*73, b) }
// func BenchmarkEncrypt740(b *testing.B) { benchmarkEncryption(1024*74, b) }
// func BenchmarkEncrypt750(b *testing.B) { benchmarkEncryption(1024*75, b) }
// func BenchmarkEncrypt760(b *testing.B) { benchmarkEncryption(1024*76, b) }
// func BenchmarkEncrypt770(b *testing.B) { benchmarkEncryption(1024*77, b) }
// func BenchmarkEncrypt780(b *testing.B) { benchmarkEncryption(1024*78, b) }
// func BenchmarkEncrypt790(b *testing.B) { benchmarkEncryption(1024*79, b) }
// func BenchmarkEncrypt800(b *testing.B) { benchmarkEncryption(1024*80, b) }

// func BenchmarkEncrypt810(b *testing.B)  { benchmarkEncryption(1024*81, b) }
// func BenchmarkEncrypt820(b *testing.B)  { benchmarkEncryption(1024*82, b) }
// func BenchmarkEncrypt830(b *testing.B)  { benchmarkEncryption(1024*83, b) }
// func BenchmarkEncrypt840(b *testing.B)  { benchmarkEncryption(1024*84, b) }
// func BenchmarkEncrypt850(b *testing.B)  { benchmarkEncryption(1024*85, b) }
// func BenchmarkEncrypt860(b *testing.B)  { benchmarkEncryption(1024*86, b) }
// func BenchmarkEncrypt870(b *testing.B)  { benchmarkEncryption(1024*87, b) }
// func BenchmarkEncrypt880(b *testing.B)  { benchmarkEncryption(1024*88, b) }
// func BenchmarkEncrypt890(b *testing.B)  { benchmarkEncryption(1024*89, b) }
// func BenchmarkEncrypt900(b *testing.B)  { benchmarkEncryption(1024*90, b) }
// func BenchmarkEncrypt910(b *testing.B)  { benchmarkEncryption(1024*91, b) }
// func BenchmarkEncrypt920(b *testing.B)  { benchmarkEncryption(1024*92, b) }
// func BenchmarkEncrypt930(b *testing.B)  { benchmarkEncryption(1024*93, b) }
// func BenchmarkEncrypt940(b *testing.B)  { benchmarkEncryption(1024*94, b) }
// func BenchmarkEncrypt950(b *testing.B)  { benchmarkEncryption(1024*95, b) }
// func BenchmarkEncrypt960(b *testing.B)  { benchmarkEncryption(1024*96, b) }
// func BenchmarkEncrypt970(b *testing.B)  { benchmarkEncryption(1024*97, b) }
// func BenchmarkEncrypt980(b *testing.B)  { benchmarkEncryption(1024*98, b) }
// func BenchmarkEncrypt990(b *testing.B)  { benchmarkEncryption(1024*99, b) }
// func BenchmarkEncrypt1000(b *testing.B) { benchmarkEncryption(1024*100, b) }

// func BenchmarkEncrypt20(b *testing.B) { benchmarkEncryption(20, b) }
// func BenchmarkEncrypt30(b *testing.B) { benchmarkEncryption(30, b) }
// func BenchmarkEncrypt40(b *testing.B) { benchmarkEncryption(40, b) }
// func BenchmarkEncrypt50(b *testing.B) { benchmarkEncryption(50, b) }
// func BenchmarkEncrypt60(b *testing.B) { benchmarkEncryption(60, b) }
// func BenchmarkEncrypt70(b *testing.B) { benchmarkEncryption(70, b) }
// func BenchmarkEncrypt80(b *testing.B) { benchmarkEncryption(80, b) }
// func BenchmarkEncrypt90(b *testing.B) { benchmarkEncryption(90, b) }

// func BenchmarkEncrypt100(b *testing.B) { benchmarkEncryption(100, b) }
// func BenchmarkEncrypt200(b *testing.B) { benchmarkEncryption(200, b) }
// func BenchmarkEncrypt300(b *testing.B) { benchmarkEncryption(300, b) }
// func BenchmarkEncrypt400(b *testing.B) { benchmarkEncryption(400, b) }
// func BenchmarkEncrypt500(b *testing.B) { benchmarkEncryption(500, b) }
// func BenchmarkEncrypt600(b *testing.B) { benchmarkEncryption(600, b) }
// func BenchmarkEncrypt700(b *testing.B) { benchmarkEncryption(700, b) }
// func BenchmarkEncrypt800(b *testing.B) { benchmarkEncryption(800, b) }
// func BenchmarkEncrypt900(b *testing.B) { benchmarkEncryption(900, b) }

// func BenchmarkEncrypt1000(b *testing.B) { benchmarkEncryption(1000, b) }
// func BenchmarkEncrypt2000(b *testing.B) { benchmarkEncryption(2000, b) }
// func BenchmarkEncrypt3000(b *testing.B) { benchmarkEncryption(3000, b) }
// func BenchmarkEncrypt4000(b *testing.B) { benchmarkEncryption(4000, b) }
// func BenchmarkEncrypt5000(b *testing.B) { benchmarkEncryption(5000, b) }
// func BenchmarkEncrypt6000(b *testing.B) { benchmarkEncryption(6000, b) }
// func BenchmarkEncrypt7000(b *testing.B) { benchmarkEncryption(7000, b) }
// func BenchmarkEncrypt8000(b *testing.B) { benchmarkEncryption(8000, b) }
// func BenchmarkEncrypt9000(b *testing.B) { benchmarkEncryption(9000, b) }

// func BenchmarkEncrypt10000(b *testing.B) { benchmarkEncryption(10000, b) }
// func BenchmarkEncrypt20000(b *testing.B) { benchmarkEncryption(20000, b) }
// func BenchmarkEncrypt30000(b *testing.B) { benchmarkEncryption(30000, b) }
// func BenchmarkEncrypt40000(b *testing.B) { benchmarkEncryption(40000, b) }
// func BenchmarkEncrypt50000(b *testing.B) { benchmarkEncryption(50000, b) }
// func BenchmarkEncrypt60000(b *testing.B) { benchmarkEncryption(60000, b) }
// func BenchmarkEncrypt70000(b *testing.B) { benchmarkEncryption(70000, b) }
// func BenchmarkEncrypt80000(b *testing.B) { benchmarkEncryption(80000, b) }
// func BenchmarkEncrypt90000(b *testing.B) { benchmarkEncryption(90000, b) }

//func BenchmarkEncrypt100000(b *testing.B) { benchmarkEncryption(100000, b) }
