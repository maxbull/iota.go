package iota

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	_ "golang.org/x/crypto/blake2b"
)

// Defines the type of signature.
type SignatureType = byte

const (
	// Denotes a WOTS signature.
	SignatureWOTS SignatureType = iota
	// Denotes an Ed25519 signature.
	SignatureEd25519

	// The size of a serialized Ed25519 signature with its type denoting byte and public key.
	Ed25519SignatureSerializedBytesSize = SmallTypeDenotationByteSize + ed25519.PublicKeySize + ed25519.SignatureSize
)

var (
	// Returned when an Ed25519 address and public key do not correspond to each other.
	ErrEd25519PubKeyAndAddrMismatch = errors.New("public key and address do not correspond to each other (Ed25519)")
	// Returned for invalid Ed25519 signatures.
	ErrEd25519SignatureInvalid = errors.New("signature is invalid (Ed25519")
)

// SignatureSelector implements SerializableSelectorFunc for signature types.
func SignatureSelector(sigType uint32) (Serializable, error) {
	var seri Serializable
	switch byte(sigType) {
	case SignatureWOTS:
		seri = &WOTSSignature{}
	case SignatureEd25519:
		seri = &Ed25519Signature{}
	default:
		return nil, fmt.Errorf("%w: type byte %d", ErrUnknownSignatureType, sigType)
	}
	return seri, nil
}

// WOTSSignature defines a WOTS signature.
type WOTSSignature struct{}

func (w *WOTSSignature) Deserialize(data []byte, deSeriMode DeSerializationMode) (int, error) {
	if deSeriMode.HasMode(DeSeriModePerformValidation) {
		if err := checkTypeByte(data, SignatureWOTS); err != nil {
			return 0, fmt.Errorf("unable to deserialize WOTS signature: %w", err)
		}
	}
	return 0, ErrWOTSNotImplemented
}

func (w *WOTSSignature) Serialize(deSeriMode DeSerializationMode) ([]byte, error) {
	return nil, ErrWOTSNotImplemented
}

func (w *WOTSSignature) MarshalJSON() ([]byte, error) {
	return nil, ErrWOTSNotImplemented
}

func (w *WOTSSignature) UnmarshalJSON(i []byte) error {
	return ErrWOTSNotImplemented
}

// Ed25519Signature defines an Ed25519 signature.
type Ed25519Signature struct {
	// The public key used to verify the given signature.
	PublicKey [ed25519.PublicKeySize]byte `json:"public_key"`
	// The signature.
	Signature [ed25519.SignatureSize]byte `json:"signature"`
}

// Valid verifies whether given the message and Ed25519 address, the signature is valid.
func (e *Ed25519Signature) Valid(msg []byte, addr *Ed25519Address) error {
	// an address is the Blake2b 256 hash of the public key
	addrFromPubKey := AddressFromEd25519PubKey(e.PublicKey[:])
	if !bytes.Equal(addr[:], addrFromPubKey[:]) {
		return fmt.Errorf("%w: address %s, public key %s", ErrEd25519PubKeyAndAddrMismatch, addr[:], addrFromPubKey)
	}
	if valid := ed25519.Verify(e.PublicKey[:], msg, e.Signature[:]); !valid {
		return fmt.Errorf("%w: address %s, public key %s, signature %s ", ErrEd25519SignatureInvalid, addr[:], e.PublicKey, e.Signature)
	}
	return nil
}

func (e *Ed25519Signature) Deserialize(data []byte, deSeriMode DeSerializationMode) (int, error) {
	if deSeriMode.HasMode(DeSeriModePerformValidation) {
		if err := checkMinByteLength(Ed25519SignatureSerializedBytesSize, len(data)); err != nil {
			return 0, fmt.Errorf("invalid Ed25519 signature bytes: %w", err)
		}
		if err := checkTypeByte(data, SignatureEd25519); err != nil {
			return 0, fmt.Errorf("unable to deserialize Ed25519 signature: %w", err)
		}
	}
	// skip type byte
	data = data[SmallTypeDenotationByteSize:]
	copy(e.PublicKey[:], data[:ed25519.PublicKeySize])
	copy(e.Signature[:], data[ed25519.PublicKeySize:])
	return Ed25519SignatureSerializedBytesSize, nil
}

func (e *Ed25519Signature) Serialize(deSeriMode DeSerializationMode) ([]byte, error) {
	var b [Ed25519SignatureSerializedBytesSize]byte
	b[0] = SignatureEd25519
	copy(b[SmallTypeDenotationByteSize:], e.PublicKey[:])
	copy(b[SmallTypeDenotationByteSize+ed25519.PublicKeySize:], e.Signature[:])
	return b[:], nil
}

func (e *Ed25519Signature) MarshalJSON() ([]byte, error) {
	jsonEdSig := &JSONEd25519Signature{}
	jsonEdSig.Type = int(SignatureEd25519)
	jsonEdSig.PublicKey = hex.EncodeToString(e.PublicKey[:])
	jsonEdSig.Signature = hex.EncodeToString(e.Signature[:])
	return json.Marshal(jsonEdSig)
}

func (e *Ed25519Signature) UnmarshalJSON(bytes []byte) error {
	jsonEdSig := &JSONEd25519Signature{}
	if err := json.Unmarshal(bytes, jsonEdSig); err != nil {
		return err
	}
	seri, err := jsonEdSig.ToSerializable()
	if err != nil {
		return err
	}
	*e = *seri.(*Ed25519Signature)
	return nil
}

// JSONSignatureSelector selects the json signature object for the given type.
func JSONSignatureSelector(ty int) (JSONSerializable, error) {
	var obj JSONSerializable
	switch byte(ty) {
	case SignatureWOTS:
		obj = &JSONWOTSSignature{}
	case SignatureEd25519:
		obj = &JSONEd25519Signature{}
	default:
		return nil, fmt.Errorf("unable to decode signature type from JSON: %w", ErrUnknownUnlockBlockType)
	}
	return obj, nil
}

// JSONEd25519Signature defines the json representation of an Ed25519Signature.
type JSONEd25519Signature struct {
	Type      int    `json:"type"`
	PublicKey string `json:"publicKey"`
	Signature string `json:"signature"`
}

func (j *JSONEd25519Signature) ToSerializable() (Serializable, error) {
	sig := &Ed25519Signature{}

	pubKeyBytes, err := hex.DecodeString(j.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("unable to decode public key from JSON for Ed25519 signature: %w", err)
	}

	sigBytes, err := hex.DecodeString(j.Signature)
	if err != nil {
		return nil, fmt.Errorf("unable to decode signature from JSON for Ed25519 signature: %w", err)
	}

	copy(sig.PublicKey[:], pubKeyBytes)
	copy(sig.Signature[:], sigBytes)
	return sig, nil
}

// JSONWOTSSignature defines the json representation of a WOTSSignature.
type JSONWOTSSignature struct {
	// TODO: implement
}

func (j *JSONWOTSSignature) ToSerializable() (Serializable, error) {
	return nil, ErrWOTSNotImplemented
}