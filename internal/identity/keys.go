// Package identity handles the cryptographic identity of YOU.
// This is the most security-critical code in QuantumLife.
package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/cloudflare/circl/kem/mlkem/mlkem768"
	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

// KeyBundle contains all cryptographic keys for an identity
type KeyBundle struct {
	// Classical keys (current standard)
	Ed25519Public  ed25519.PublicKey
	Ed25519Private ed25519.PrivateKey

	// Post-quantum signing (ML-DSA-65, FIPS 204)
	MLDSAPublic  mldsa65.PublicKey
	MLDSAPrivate mldsa65.PrivateKey

	// Post-quantum key encapsulation (ML-KEM-768, FIPS 203)
	MLKEMPublic  mlkem768.PublicKey
	MLKEMPrivate mlkem768.PrivateKey
}

// SerializedKeyBundle is the encrypted, storable form of keys
type SerializedKeyBundle struct {
	// Public keys (stored as base64, not encrypted)
	Ed25519Public string `json:"ed25519_public"`
	MLDSAPublic   string `json:"mldsa_public"`
	MLKEMPublic   string `json:"mlkem_public"`

	// Private keys (encrypted with passphrase)
	EncryptedPrivateKeys string `json:"encrypted_private_keys"`

	// Key derivation parameters
	Salt      string `json:"salt"`      // Base64 encoded
	Algorithm string `json:"algorithm"` // "argon2id"
}

// GenerateKeyBundle creates a complete set of cryptographic keys.
// This is called once when creating a new identity.
func GenerateKeyBundle() (*KeyBundle, error) {
	bundle := &KeyBundle{}

	// Generate Ed25519 keys (classical signing)
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Ed25519 keys: %w", err)
	}
	bundle.Ed25519Public = pub
	bundle.Ed25519Private = priv

	// Generate ML-DSA-65 keys (post-quantum signing)
	mldsaPub, mldsaPriv, err := mldsa65.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ML-DSA keys: %w", err)
	}
	bundle.MLDSAPublic = *mldsaPub
	bundle.MLDSAPrivate = *mldsaPriv

	// Generate ML-KEM-768 keys (post-quantum key encapsulation)
	mlkemPub, mlkemPriv, err := mlkem768.GenerateKeyPair(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ML-KEM keys: %w", err)
	}
	bundle.MLKEMPublic = *mlkemPub
	bundle.MLKEMPrivate = *mlkemPriv

	return bundle, nil
}

// Serialize encrypts and serializes the key bundle for storage.
// The passphrase is used to derive an encryption key via Argon2id.
func (kb *KeyBundle) Serialize(passphrase string) (*SerializedKeyBundle, error) {
	// Generate salt for key derivation
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive encryption key using Argon2id
	key := argon2.IDKey([]byte(passphrase), salt, 3, 64*1024, 4, 32)

	// Create AEAD cipher (XChaCha20-Poly1305)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Serialize private keys
	privateData := serializePrivateKeys(kb)

	// Generate nonce
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt private keys
	encrypted := aead.Seal(nonce, nonce, privateData, nil)

	// Serialize public keys
	mldsaPubBytes, _ := kb.MLDSAPublic.MarshalBinary()
	mlkemPubBytes, _ := kb.MLKEMPublic.MarshalBinary()

	return &SerializedKeyBundle{
		Ed25519Public:        base64.StdEncoding.EncodeToString(kb.Ed25519Public),
		MLDSAPublic:          base64.StdEncoding.EncodeToString(mldsaPubBytes),
		MLKEMPublic:          base64.StdEncoding.EncodeToString(mlkemPubBytes),
		EncryptedPrivateKeys: base64.StdEncoding.EncodeToString(encrypted),
		Salt:                 base64.StdEncoding.EncodeToString(salt),
		Algorithm:            "argon2id",
	}, nil
}

// Deserialize decrypts and reconstructs the key bundle.
func (skb *SerializedKeyBundle) Deserialize(passphrase string) (*KeyBundle, error) {
	// Decode salt
	salt, err := base64.StdEncoding.DecodeString(skb.Salt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	// Derive encryption key
	key := argon2.IDKey([]byte(passphrase), salt, 3, 64*1024, 4, 32)

	// Create AEAD cipher
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Decode encrypted private keys
	encrypted, err := base64.StdEncoding.DecodeString(skb.EncryptedPrivateKeys)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted keys: %w", err)
	}

	// Decrypt
	if len(encrypted) < aead.NonceSize() {
		return nil, errors.New("invalid encrypted data")
	}
	nonce := encrypted[:aead.NonceSize()]
	ciphertext := encrypted[aead.NonceSize():]

	privateData, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (wrong passphrase?): %w", err)
	}

	// Deserialize private keys
	bundle, err := deserializePrivateKeys(privateData)
	if err != nil {
		return nil, err
	}

	// Decode and set public keys
	ed25519Pub, err := base64.StdEncoding.DecodeString(skb.Ed25519Public)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Ed25519 public key: %w", err)
	}
	bundle.Ed25519Public = ed25519Pub

	mldsaPubBytes, err := base64.StdEncoding.DecodeString(skb.MLDSAPublic)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ML-DSA public key: %w", err)
	}
	mldsaPub := new(mldsa65.PublicKey)
	if err := mldsaPub.UnmarshalBinary(mldsaPubBytes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ML-DSA public key: %w", err)
	}
	bundle.MLDSAPublic = *mldsaPub

	mlkemPubBytes, err := base64.StdEncoding.DecodeString(skb.MLKEMPublic)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ML-KEM public key: %w", err)
	}
	mlkemPub := new(mlkem768.PublicKey)
	if err := mlkemPub.Unpack(mlkemPubBytes); err != nil {
		return nil, fmt.Errorf("failed to unpack ML-KEM public key: %w", err)
	}
	bundle.MLKEMPublic = *mlkemPub

	return bundle, nil
}

// serializePrivateKeys packs private keys into bytes
func serializePrivateKeys(kb *KeyBundle) []byte {
	// Format: [ed25519_len:4][ed25519][mldsa_len:4][mldsa][mlkem_len:4][mlkem]
	ed25519Bytes := []byte(kb.Ed25519Private)
	mldsaBytes, _ := kb.MLDSAPrivate.MarshalBinary()
	mlkemBytes, _ := kb.MLKEMPrivate.MarshalBinary()

	total := 12 + len(ed25519Bytes) + len(mldsaBytes) + len(mlkemBytes)
	buf := make([]byte, total)

	offset := 0

	// Ed25519
	writeLen(buf[offset:], len(ed25519Bytes))
	offset += 4
	copy(buf[offset:], ed25519Bytes)
	offset += len(ed25519Bytes)

	// ML-DSA
	writeLen(buf[offset:], len(mldsaBytes))
	offset += 4
	copy(buf[offset:], mldsaBytes)
	offset += len(mldsaBytes)

	// ML-KEM
	writeLen(buf[offset:], len(mlkemBytes))
	offset += 4
	copy(buf[offset:], mlkemBytes)

	return buf
}

// deserializePrivateKeys unpacks private keys from bytes
func deserializePrivateKeys(data []byte) (*KeyBundle, error) {
	bundle := &KeyBundle{}
	offset := 0

	// Ed25519
	if offset+4 > len(data) {
		return nil, errors.New("invalid private key data: too short for Ed25519 length")
	}
	ed25519Len := readLen(data[offset:])
	offset += 4
	if offset+ed25519Len > len(data) {
		return nil, errors.New("invalid private key data: too short for Ed25519 key")
	}
	bundle.Ed25519Private = make(ed25519.PrivateKey, ed25519Len)
	copy(bundle.Ed25519Private, data[offset:offset+ed25519Len])
	offset += ed25519Len

	// ML-DSA
	if offset+4 > len(data) {
		return nil, errors.New("invalid private key data: too short for ML-DSA length")
	}
	mldsaLen := readLen(data[offset:])
	offset += 4
	if offset+mldsaLen > len(data) {
		return nil, errors.New("invalid private key data: too short for ML-DSA key")
	}
	mldsaPriv := new(mldsa65.PrivateKey)
	if err := mldsaPriv.UnmarshalBinary(data[offset : offset+mldsaLen]); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ML-DSA key: %w", err)
	}
	bundle.MLDSAPrivate = *mldsaPriv
	offset += mldsaLen

	// ML-KEM
	if offset+4 > len(data) {
		return nil, errors.New("invalid private key data: too short for ML-KEM length")
	}
	mlkemLen := readLen(data[offset:])
	offset += 4
	if offset+mlkemLen > len(data) {
		return nil, errors.New("invalid private key data: too short for ML-KEM key")
	}
	mlkemPriv := new(mlkem768.PrivateKey)
	if err := mlkemPriv.Unpack(data[offset : offset+mlkemLen]); err != nil {
		return nil, fmt.Errorf("failed to unpack ML-KEM key: %w", err)
	}
	bundle.MLKEMPrivate = *mlkemPriv

	return bundle, nil
}

func writeLen(buf []byte, length int) {
	buf[0] = byte(length >> 24)
	buf[1] = byte(length >> 16)
	buf[2] = byte(length >> 8)
	buf[3] = byte(length)
}

func readLen(buf []byte) int {
	return int(buf[0])<<24 | int(buf[1])<<16 | int(buf[2])<<8 | int(buf[3])
}

// -----------------------------------------------------------------------------
// Signing operations
// -----------------------------------------------------------------------------

// SignHybrid signs data with both Ed25519 and ML-DSA-65 (hybrid signature)
func (kb *KeyBundle) SignHybrid(data []byte) (ed25519Sig, mldsaSig []byte, err error) {
	ed25519Sig = ed25519.Sign(kb.Ed25519Private, data)

	// ML-DSA-65 signature
	mldsaSig = make([]byte, mldsa65.SignatureSize)
	if err := mldsa65.SignTo(&kb.MLDSAPrivate, data, nil, false, mldsaSig); err != nil {
		return nil, nil, fmt.Errorf("ML-DSA signing failed: %w", err)
	}

	return ed25519Sig, mldsaSig, nil
}

// VerifyHybrid verifies both signatures
func (kb *KeyBundle) VerifyHybrid(data, ed25519Sig, mldsaSig []byte) bool {
	ed25519Valid := ed25519.Verify(kb.Ed25519Public, data, ed25519Sig)
	mldsaValid := mldsa65.Verify(&kb.MLDSAPublic, data, nil, mldsaSig)
	return ed25519Valid && mldsaValid
}

// -----------------------------------------------------------------------------
// Key encapsulation (for establishing shared secrets)
// -----------------------------------------------------------------------------

// SharedSecretSize is the size of the shared secret in bytes
const SharedSecretSize = 32

// CiphertextSize is the size of the ML-KEM-768 ciphertext
const CiphertextSize = 1088

// Encapsulate creates a shared secret for the recipient's public key
// Returns: ciphertext (to send), sharedSecret (to use for encryption)
func Encapsulate(recipientPublicKey *mlkem768.PublicKey) (ciphertext, sharedSecret []byte, err error) {
	ct := make([]byte, CiphertextSize)
	ss := make([]byte, SharedSecretSize)

	// Generate random seed for encapsulation
	seed := make([]byte, 32)
	if _, err := rand.Read(seed); err != nil {
		return nil, nil, fmt.Errorf("failed to generate seed: %w", err)
	}

	recipientPublicKey.EncapsulateTo(ct, ss, seed)
	return ct, ss, nil
}

// Decapsulate recovers the shared secret from a ciphertext
func (kb *KeyBundle) Decapsulate(ciphertext []byte) (sharedSecret []byte, err error) {
	ss := make([]byte, SharedSecretSize)
	kb.MLKEMPrivate.DecapsulateTo(ss, ciphertext)
	return ss, nil
}
