package identity

import (
	"bytes"
	"testing"
)

func TestGenerateKeyBundle(t *testing.T) {
	bundle, err := GenerateKeyBundle()
	if err != nil {
		t.Fatalf("GenerateKeyBundle failed: %v", err)
	}

	if bundle == nil {
		t.Fatal("bundle is nil")
	}

	// Check Ed25519 keys
	if bundle.Ed25519Public == nil {
		t.Error("Ed25519Public is nil")
	}
	if bundle.Ed25519Private == nil {
		t.Error("Ed25519Private is nil")
	}

	// Check ML-DSA keys (struct values, check if valid by serializing)
	mldsaPub, err := bundle.MLDSAPublic.MarshalBinary()
	if err != nil || len(mldsaPub) == 0 {
		t.Error("MLDSAPublic not valid")
	}

	mldsaPriv, err := bundle.MLDSAPrivate.MarshalBinary()
	if err != nil || len(mldsaPriv) == 0 {
		t.Error("MLDSAPrivate not valid")
	}

	// Check ML-KEM keys
	mlkemPub, err := bundle.MLKEMPublic.MarshalBinary()
	if err != nil || len(mlkemPub) == 0 {
		t.Error("MLKEMPublic not valid")
	}

	mlkemPriv, err := bundle.MLKEMPrivate.MarshalBinary()
	if err != nil || len(mlkemPriv) == 0 {
		t.Error("MLKEMPrivate not valid")
	}
}

func TestKeyBundle_SerializeDeserialize(t *testing.T) {
	bundle, err := GenerateKeyBundle()
	if err != nil {
		t.Fatalf("GenerateKeyBundle failed: %v", err)
	}

	passphrase := "test-passphrase-123"

	// Serialize
	serialized, err := bundle.Serialize(passphrase)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	if serialized == nil {
		t.Fatal("serialized is nil")
	}
	if serialized.Ed25519Public == "" {
		t.Error("Ed25519Public empty")
	}
	if serialized.MLDSAPublic == "" {
		t.Error("MLDSAPublic empty")
	}
	if serialized.MLKEMPublic == "" {
		t.Error("MLKEMPublic empty")
	}
	if serialized.EncryptedPrivateKeys == "" {
		t.Error("EncryptedPrivateKeys empty")
	}
	if serialized.Salt == "" {
		t.Error("Salt empty")
	}
	if serialized.Algorithm != "argon2id" {
		t.Errorf("Algorithm = %v, want argon2id", serialized.Algorithm)
	}

	// Deserialize
	restored, err := serialized.Deserialize(passphrase)
	if err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	// Compare keys
	if !bytes.Equal(bundle.Ed25519Public, restored.Ed25519Public) {
		t.Error("Ed25519Public mismatch")
	}
	if !bytes.Equal(bundle.Ed25519Private, restored.Ed25519Private) {
		t.Error("Ed25519Private mismatch")
	}
}

func TestKeyBundle_Deserialize_WrongPassphrase(t *testing.T) {
	bundle, err := GenerateKeyBundle()
	if err != nil {
		t.Fatalf("GenerateKeyBundle failed: %v", err)
	}

	serialized, err := bundle.Serialize("correct-passphrase")
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	_, err = serialized.Deserialize("wrong-passphrase")
	if err == nil {
		t.Error("expected error with wrong passphrase")
	}
}

func TestKeyBundle_Deserialize_InvalidSalt(t *testing.T) {
	serialized := &SerializedKeyBundle{
		Salt: "invalid-base64!!!",
	}

	_, err := serialized.Deserialize("password")
	if err == nil {
		t.Error("expected error with invalid salt")
	}
}

func TestKeyBundle_Deserialize_InvalidEncryptedKeys(t *testing.T) {
	serialized := &SerializedKeyBundle{
		Salt:                 "dGVzdHNhbHQ=", // "testsalt" in base64
		EncryptedPrivateKeys: "invalid-base64!!!",
	}

	_, err := serialized.Deserialize("password")
	if err == nil {
		t.Error("expected error with invalid encrypted keys")
	}
}

func TestKeyBundle_Deserialize_TooShortEncryptedData(t *testing.T) {
	serialized := &SerializedKeyBundle{
		Salt:                 "dGVzdHNhbHQ=",
		EncryptedPrivateKeys: "YWJj", // "abc" - too short for nonce
	}

	_, err := serialized.Deserialize("password")
	if err == nil {
		t.Error("expected error with too short encrypted data")
	}
}

func TestKeyBundle_SignHybrid(t *testing.T) {
	bundle, err := GenerateKeyBundle()
	if err != nil {
		t.Fatalf("GenerateKeyBundle failed: %v", err)
	}

	data := []byte("test message to sign")

	ed25519Sig, mldsaSig, err := bundle.SignHybrid(data)
	if err != nil {
		t.Fatalf("SignHybrid failed: %v", err)
	}

	if len(ed25519Sig) == 0 {
		t.Error("ed25519 signature empty")
	}
	if len(mldsaSig) == 0 {
		t.Error("mldsa signature empty")
	}
}

func TestKeyBundle_VerifyHybrid(t *testing.T) {
	bundle, err := GenerateKeyBundle()
	if err != nil {
		t.Fatalf("GenerateKeyBundle failed: %v", err)
	}

	data := []byte("test message to sign")

	ed25519Sig, mldsaSig, err := bundle.SignHybrid(data)
	if err != nil {
		t.Fatalf("SignHybrid failed: %v", err)
	}

	// Verify valid signatures
	valid := bundle.VerifyHybrid(data, ed25519Sig, mldsaSig)
	if !valid {
		t.Error("valid signatures should verify")
	}

	// Verify with modified data
	valid = bundle.VerifyHybrid([]byte("modified data"), ed25519Sig, mldsaSig)
	if valid {
		t.Error("should fail with modified data")
	}

	// Verify with modified signature
	modifiedSig := make([]byte, len(ed25519Sig))
	copy(modifiedSig, ed25519Sig)
	modifiedSig[0] ^= 0xFF
	valid = bundle.VerifyHybrid(data, modifiedSig, mldsaSig)
	if valid {
		t.Error("should fail with modified signature")
	}
}

func TestEncapsulateDecapsulate(t *testing.T) {
	bundle, err := GenerateKeyBundle()
	if err != nil {
		t.Fatalf("GenerateKeyBundle failed: %v", err)
	}

	// Encapsulate to bundle's public key
	ciphertext, sharedSecret1, err := Encapsulate(&bundle.MLKEMPublic)
	if err != nil {
		t.Fatalf("Encapsulate failed: %v", err)
	}

	if len(ciphertext) != CiphertextSize {
		t.Errorf("ciphertext size = %d, want %d", len(ciphertext), CiphertextSize)
	}
	if len(sharedSecret1) != SharedSecretSize {
		t.Errorf("shared secret size = %d, want %d", len(sharedSecret1), SharedSecretSize)
	}

	// Decapsulate
	sharedSecret2, err := bundle.Decapsulate(ciphertext)
	if err != nil {
		t.Fatalf("Decapsulate failed: %v", err)
	}

	// Shared secrets should match
	if !bytes.Equal(sharedSecret1, sharedSecret2) {
		t.Error("shared secrets do not match")
	}
}

func TestEncryptDecryptWithKey(t *testing.T) {
	bundle, err := GenerateKeyBundle()
	if err != nil {
		t.Fatalf("GenerateKeyBundle failed: %v", err)
	}

	original := []byte("secret data to encrypt")

	// Encrypt
	encrypted, err := encryptWithKey(bundle, original)
	if err != nil {
		t.Fatalf("encryptWithKey failed: %v", err)
	}

	// Decrypt
	decrypted, err := decryptWithKey(bundle, encrypted)
	if err != nil {
		t.Fatalf("decryptWithKey failed: %v", err)
	}

	if !bytes.Equal(original, decrypted) {
		t.Error("decrypted data does not match original")
	}
}

func TestDecryptWithKey_TooShort(t *testing.T) {
	bundle, err := GenerateKeyBundle()
	if err != nil {
		t.Fatalf("GenerateKeyBundle failed: %v", err)
	}

	// Too short for nonce
	_, err = decryptWithKey(bundle, []byte("short"))
	if err == nil {
		t.Error("expected error for too short ciphertext")
	}
}

func TestDecryptWithKey_InvalidCiphertext(t *testing.T) {
	bundle, err := GenerateKeyBundle()
	if err != nil {
		t.Fatalf("GenerateKeyBundle failed: %v", err)
	}

	// Valid length but invalid ciphertext
	invalid := make([]byte, 100)
	_, err = decryptWithKey(bundle, invalid)
	if err == nil {
		t.Error("expected error for invalid ciphertext")
	}
}

func TestDeriveSymmetricKey(t *testing.T) {
	seed := []byte("test-seed-32-bytes-long-here!!!")

	key := deriveSymmetricKey(seed)

	if len(key) != 32 {
		t.Errorf("key length = %d, want 32", len(key))
	}

	// Same seed should give same key
	key2 := deriveSymmetricKey(seed)
	if !bytes.Equal(key, key2) {
		t.Error("same seed should give same key")
	}

	// Different seed should give different key
	differentSeed := []byte("different-seed-32bytes-here!!!!")
	key3 := deriveSymmetricKey(differentSeed)
	if bytes.Equal(key, key3) {
		t.Error("different seeds should give different keys")
	}
}

func TestWriteReadLen(t *testing.T) {
	tests := []int{0, 1, 255, 256, 65535, 1000000}

	for _, length := range tests {
		buf := make([]byte, 4)
		writeLen(buf, length)
		result := readLen(buf)

		if result != length {
			t.Errorf("writeLen/readLen(%d) = %d", length, result)
		}
	}
}

func TestSerializeDeserializePrivateKeys(t *testing.T) {
	bundle, err := GenerateKeyBundle()
	if err != nil {
		t.Fatalf("GenerateKeyBundle failed: %v", err)
	}

	// Serialize
	serialized := serializePrivateKeys(bundle)

	// Deserialize
	restored, err := deserializePrivateKeys(serialized)
	if err != nil {
		t.Fatalf("deserializePrivateKeys failed: %v", err)
	}

	// Compare Ed25519 private key
	if !bytes.Equal(bundle.Ed25519Private, restored.Ed25519Private) {
		t.Error("Ed25519Private mismatch")
	}

	// Compare by serializing ML-DSA keys
	origMldsa, _ := bundle.MLDSAPrivate.MarshalBinary()
	restoredMldsa, _ := restored.MLDSAPrivate.MarshalBinary()
	if !bytes.Equal(origMldsa, restoredMldsa) {
		t.Error("MLDSAPrivate mismatch")
	}

	// Compare by serializing ML-KEM keys
	origMlkem, _ := bundle.MLKEMPrivate.MarshalBinary()
	restoredMlkem, _ := restored.MLKEMPrivate.MarshalBinary()
	if !bytes.Equal(origMlkem, restoredMlkem) {
		t.Error("MLKEMPrivate mismatch")
	}
}

func TestDeserializePrivateKeys_InvalidData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"too short for ed25519 length", []byte{0, 0}},
		{"invalid ed25519 length", []byte{255, 255, 255, 255}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := deserializePrivateKeys(tt.data)
			if err == nil {
				t.Error("expected error for invalid data")
			}
		})
	}
}

func TestConstants(t *testing.T) {
	if SharedSecretSize != 32 {
		t.Errorf("SharedSecretSize = %d, want 32", SharedSecretSize)
	}
	if CiphertextSize != 1088 {
		t.Errorf("CiphertextSize = %d, want 1088", CiphertextSize)
	}
}

func TestSerializedKeyBundle_Fields(t *testing.T) {
	skb := SerializedKeyBundle{
		Ed25519Public:        "ed25519-pub",
		MLDSAPublic:          "mldsa-pub",
		MLKEMPublic:          "mlkem-pub",
		EncryptedPrivateKeys: "encrypted",
		Salt:                 "salt",
		Algorithm:            "argon2id",
	}

	if skb.Ed25519Public != "ed25519-pub" {
		t.Error("Ed25519Public not set correctly")
	}
	if skb.MLDSAPublic != "mldsa-pub" {
		t.Error("MLDSAPublic not set correctly")
	}
	if skb.MLKEMPublic != "mlkem-pub" {
		t.Error("MLKEMPublic not set correctly")
	}
	if skb.Algorithm != "argon2id" {
		t.Error("Algorithm not set correctly")
	}
}

func TestKeyBundleUniqueness(t *testing.T) {
	// Generate two bundles, ensure they're different
	bundle1, err := GenerateKeyBundle()
	if err != nil {
		t.Fatalf("GenerateKeyBundle 1 failed: %v", err)
	}

	bundle2, err := GenerateKeyBundle()
	if err != nil {
		t.Fatalf("GenerateKeyBundle 2 failed: %v", err)
	}

	// Ed25519 keys should be different
	if bytes.Equal(bundle1.Ed25519Public, bundle2.Ed25519Public) {
		t.Error("two generated bundles should have different Ed25519 public keys")
	}

	if bytes.Equal(bundle1.Ed25519Private, bundle2.Ed25519Private) {
		t.Error("two generated bundles should have different Ed25519 private keys")
	}
}
