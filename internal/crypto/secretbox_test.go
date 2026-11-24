package crypto

import (
	"bytes"
	"testing"
)

func TestSecretBoxRoundTrip(t *testing.T) {
	box, err := New(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	plain := []byte("client-secret-value")
	cipherText, err := box.Seal(plain)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if bytes.Contains(cipherText, plain) {
		t.Fatal("密文中不应包含明文")
	}
	got, err := box.Open(cipherText)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("got %q", got)
	}
}

func TestSecretBoxRejectsTamper(t *testing.T) {
	box, _ := New(bytes.Repeat([]byte{1}, 32))
	cipherText, _ := box.Seal([]byte("x"))
	cipherText[len(cipherText)-1] ^= 0xff
	if _, err := box.Open(cipherText); err == nil {
		t.Fatal("篡改后的密文应解密失败")
	}
}

func TestSecretBoxRejectsWrongKey(t *testing.T) {
	box1, _ := New(bytes.Repeat([]byte{1}, 32))
	box2, _ := New(bytes.Repeat([]byte{2}, 32))
	cipherText, _ := box1.Seal([]byte("x"))
	if _, err := box2.Open(cipherText); err == nil {
		t.Fatal("错误密钥应解密失败")
	}
}

func TestNewRejectsBadKey(t *testing.T) {
	if _, err := New([]byte("short")); err == nil {
		t.Fatal("短密钥应失败")
	}
}
