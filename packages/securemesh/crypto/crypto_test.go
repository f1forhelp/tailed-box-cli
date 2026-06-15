package crypto

import "testing"

func TestRandomBytesLength(t *testing.T) {
	buf, err := RandomBytes(32)
	if err != nil {
		t.Fatalf("RandomBytes: %v", err)
	}
	if len(buf) != 32 {
		t.Fatalf("len = %d, want 32", len(buf))
	}
}

func TestEncodingRoundTrip(t *testing.T) {
	input := []byte("secure mesh")
	encoded := Base32NoPaddingEncode(input)
	decoded, err := Base32NoPaddingDecode(encoded[:4] + "-" + encoded[4:])
	if err != nil {
		t.Fatalf("Base32NoPaddingDecode: %v", err)
	}
	if string(decoded) != string(input) {
		t.Fatalf("decoded = %q, want %q", decoded, input)
	}
}

func TestConstantTimeEqual(t *testing.T) {
	if !ConstantTimeEqual([]byte("same"), []byte("same")) {
		t.Fatal("equal values were not equal")
	}
	if ConstantTimeEqual([]byte("same"), []byte("diff")) {
		t.Fatal("different values were equal")
	}
	if ConstantTimeEqual([]byte("same"), []byte("same-longer")) {
		t.Fatal("different lengths were equal")
	}
}
