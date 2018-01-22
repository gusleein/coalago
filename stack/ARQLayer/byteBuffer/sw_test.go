package byteBuffer

import (
	"testing"
)

func TestNextFrame(t *testing.T) {
	str := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	buffer := NewBuffer()
	buffer.Write([]byte(str))

	var outputStr string
	for {
		_, frame, next := buffer.NextFrame()
		outputStr += string(frame.Body)
		if !next {
			break
		}
	}

	if outputStr != str {
		t.Error("Invalid output")
	}
}

func TestWindow(t *testing.T) {
	winSize := 3
	str := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffgggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggghhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiijjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkk"
	buffer := NewBuffer()
	buffer.Write([]byte(str))

	var outputStr string
	first := true
	for {
		window, last := buffer.Window(winSize)
		if first {
			for _, frame := range window {
				outputStr += string(frame.Body)
			}
		} else {
			outputStr += string(window[len(window)-1].Body)
		}
		first = false

		if last {
			break
		}
	}

	if outputStr != str {
		t.Error("Invalid output: ", outputStr)
	}

	t.Log("Get window is successful")

	// Check with next ////////////////////////////////

	buffer = NewBuffer()
	buffer.Write([]byte(str))
	outputStr = ""

	for {
		_, frame, next := buffer.NextFrame()
		if frame == nil {
			break
		}
		outputStr += string(frame.Body)
		if !next {
			break
		}
	}

	if outputStr != str {
		t.Error("Invalid output: ", outputStr)
	}
}

func TestWrite(t *testing.T) {
	str := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffgggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggghhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiijjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkk"
	buffer := NewBuffer()
	buffer.Write([]byte(str))
	if buffer.count != len(str)/64 {
		t.Error("Invalid count: ", buffer.count)
	}

	outputStr := string(buffer.Byte())
	if outputStr != str {
		t.Error("Invalid output: ", outputStr)
	}
}

func TestWriteToWindow(t *testing.T) {
	str := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffgggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggghhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiijjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkk"
	buffer := NewBuffer()
	buffer.SetLastBlock(11)

	index := 0
	for i := 0; i < len(str); i += 64 {
		chunk := str[i : i+64]
		isFull, err := buffer.WriteToWindow(3, index, []byte(chunk))
		if err != nil {
			t.Error(err)
		}

		if isFull {
			break
		}
		index++
	}

	if buffer.count != len(str)/64 {
		t.Error("Invalid count: ", buffer.count)
	}

	outputStr := string(buffer.Byte())
	if outputStr != str {
		t.Error("Invalid output: ", outputStr)
	}
}
