package websocket

import "testing"

func TestSecWebsocketAccept(t *testing.T) {
	input := "dGhlIHNhbXBsZSBub25jZQ=="

	actual := newSecWebsocketAccept(input)

	expected := "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="
	if actual.String() != expected {
		t.Errorf("NewSecWebsocketAccept(%q) = %q, expected %q", input, actual, expected)
	} else {
		t.Logf("NewSecWebsocketAccept(%q) = %q, OK", input, actual)
	}
}
