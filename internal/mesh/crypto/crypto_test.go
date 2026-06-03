package meshcrypto

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

func TestTranscriptSignatureVerification(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate identity key: %v", err)
	}
	transcript := testTranscript(t)

	signature, err := SignTranscript(privateKey, transcript)
	if err != nil {
		t.Fatalf("sign transcript: %v", err)
	}
	if !VerifyTranscript(publicKey, transcript, signature) {
		t.Fatal("expected signature to verify")
	}

	transcript.ResponderNodeID = "node_other"
	if VerifyTranscript(publicKey, transcript, signature) {
		t.Fatal("signature verified after transcript tampering")
	}
}

func TestDeriveSessionKeysMatchBothSides(t *testing.T) {
	initiatorPrivate, initiatorPublic, err := GenerateEphemeral()
	if err != nil {
		t.Fatalf("generate initiator key: %v", err)
	}
	responderPrivate, responderPublic, err := GenerateEphemeral()
	if err != nil {
		t.Fatalf("generate responder key: %v", err)
	}
	transcript := testTranscriptWithKeys(initiatorPublic, responderPublic)

	initiatorKeys, err := DeriveSessionKeys(initiatorPrivate, responderPublic, transcript)
	if err != nil {
		t.Fatalf("derive initiator keys: %v", err)
	}
	responderKeys, err := DeriveSessionKeys(responderPrivate, initiatorPublic, transcript)
	if err != nil {
		t.Fatalf("derive responder keys: %v", err)
	}
	if !bytes.Equal(initiatorKeys.InitiatorToResponderKey, responderKeys.InitiatorToResponderKey) {
		t.Fatal("initiator-to-responder keys differ")
	}
	if !bytes.Equal(initiatorKeys.ResponderToInitiatorKey, responderKeys.ResponderToInitiatorKey) {
		t.Fatal("responder-to-initiator keys differ")
	}
	if bytes.Equal(initiatorKeys.InitiatorToResponderKey, initiatorKeys.ResponderToInitiatorKey) {
		t.Fatal("directional keys should differ")
	}
	if len(initiatorKeys.InitiatorNoncePrefix) != NoncePrefixLen || len(initiatorKeys.ResponderNoncePrefix) != NoncePrefixLen {
		t.Fatalf("unexpected nonce prefix lengths: %#v", initiatorKeys)
	}
	if len(initiatorKeys.TranscriptHash) != 32 {
		t.Fatalf("transcript hash length = %d, want 32", len(initiatorKeys.TranscriptHash))
	}
}

func TestAEADRoundTripUsesAssociatedData(t *testing.T) {
	initiatorPrivate, initiatorPublic, err := GenerateEphemeral()
	if err != nil {
		t.Fatalf("generate initiator key: %v", err)
	}
	_, responderPublic, err := GenerateEphemeral()
	if err != nil {
		t.Fatalf("generate responder key: %v", err)
	}
	keys, err := DeriveSessionKeys(initiatorPrivate, responderPublic, testTranscriptWithKeys(initiatorPublic, responderPublic))
	if err != nil {
		t.Fatalf("derive keys: %v", err)
	}
	aead, err := NewAEAD(keys.InitiatorToResponderKey)
	if err != nil {
		t.Fatalf("new aead: %v", err)
	}
	nonce, err := Nonce(keys.InitiatorNoncePrefix, 7)
	if err != nil {
		t.Fatalf("nonce: %v", err)
	}
	associatedData := []byte("packet-header")
	ciphertext := aead.Seal(nil, nonce, []byte("ping"), associatedData)

	plaintext, err := aead.Open(nil, nonce, ciphertext, associatedData)
	if err != nil {
		t.Fatalf("open ciphertext: %v", err)
	}
	if !bytes.Equal(plaintext, []byte("ping")) {
		t.Fatalf("plaintext = %q", plaintext)
	}
	if _, err := aead.Open(nil, nonce, ciphertext, []byte("wrong-header")); err == nil {
		t.Fatal("expected associated-data mismatch to fail")
	}
}

func TestNonceLayout(t *testing.T) {
	nonce, err := Nonce([]byte{1, 2, 3, 4}, 9)
	if err != nil {
		t.Fatalf("nonce: %v", err)
	}
	if len(nonce) != NonceLen {
		t.Fatalf("nonce length = %d, want %d", len(nonce), NonceLen)
	}
	if !bytes.Equal(nonce[:4], []byte{1, 2, 3, 4}) {
		t.Fatalf("nonce prefix = %v", nonce[:4])
	}
	if !bytes.Equal(nonce[4:], []byte{0, 0, 0, 0, 0, 0, 0, 9}) {
		t.Fatalf("nonce sequence bytes = %v", nonce[4:])
	}
}

func testTranscript(t *testing.T) Transcript {
	t.Helper()
	_, initiatorPublic, err := GenerateEphemeral()
	if err != nil {
		t.Fatalf("generate initiator key: %v", err)
	}
	_, responderPublic, err := GenerateEphemeral()
	if err != nil {
		t.Fatalf("generate responder key: %v", err)
	}
	return testTranscriptWithKeys(initiatorPublic, responderPublic)
}

func testTranscriptWithKeys(initiatorPublic, responderPublic []byte) Transcript {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	return Transcript{
		ClusterID:                    "cluster_1",
		InitiatorNodeID:              "node_worker",
		InitiatorRole:                "worker",
		InitiatorIdentityFingerprint: "tbx1_worker",
		InitiatorEphemeralPublic:     initiatorPublic,
		InitiatorNonce:               []byte("initiator nonce"),
		InitiatorTime:                now,
		ResponderNodeID:              "node_master",
		ResponderRole:                "master",
		ResponderIdentityFingerprint: "tbx1_master",
		ResponderEphemeralPublic:     responderPublic,
		ResponderNonce:               []byte("responder nonce"),
		ResponderTime:                now.Add(time.Second),
	}
}
