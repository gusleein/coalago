package message

import "testing"

func TestParseQuery(t *testing.T) {
	m := ParseQuery("a=hello&b=War %26 World")

	if m["a"][0] != "hello" {
		t.Error("Invalid parse query")
	}
	if m["b"][0] != "War & World" {
		t.Error("Invalid parse query")
	}
}

func TestEscapeString(t *testing.T) {
	s := EscapeString("Hello & Hi")
	if s != "Hello %26 Hi" {
		t.Error("Invalid escaping:", s)
	}
}
