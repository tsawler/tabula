package core

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rc4"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
)

// ErrEncryptedNeedsPassword is returned when a PDF is encrypted and neither the
// empty user password nor the empty owner password unlocks it.
var ErrEncryptedNeedsPassword = errors.New("pdf is encrypted and requires a password")

// passwordPad is the 32-byte padding string from the PDF standard security
// handler (ISO 32000-1, 7.6.3.3).
var passwordPad = []byte{
	0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41, 0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08,
	0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80, 0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A,
}

type cryptMethod int

const (
	cryptIdentity cryptMethod = iota // no encryption (Identity crypt filter)
	cryptRC4
	cryptAESV2 // AES-128-CBC
	cryptAESV3 // AES-256-CBC
)

// StdSecurityHandler decrypts strings and streams in a PDF that uses the
// standard security handler. Only the empty user/owner password is supported.
type StdSecurityHandler struct {
	key             []byte // file encryption key
	stmMethod       cryptMethod
	strMethod       cryptMethod
	encryptMetadata bool
}

// EncryptMetadata reports whether the document's Metadata stream is encrypted.
func (h *StdSecurityHandler) EncryptMetadata() bool { return h.encryptMetadata }

// NewStdSecurityHandler builds a handler from the /Encrypt dictionary and the
// first element of the document /ID. It derives the file key for the empty
// password, falling back from user to owner.
func NewStdSecurityHandler(enc Dict, id []byte) (*StdSecurityHandler, error) {
	if name, _ := enc.Get("Filter").(Name); string(name) != "Standard" {
		return nil, fmt.Errorf("unsupported security handler: %v", enc.Get("Filter"))
	}

	v := intDefault(enc, "V", 0)
	r := intDefault(enc, "R", 0)
	length := intDefault(enc, "Length", 40)
	p := int32(intDefault(enc, "P", 0))
	o := stringBytes(enc, "O")
	u := stringBytes(enc, "U")
	encMeta := boolDefault(enc, "EncryptMetadata", true)

	h := &StdSecurityHandler{encryptMetadata: encMeta}

	// Determine the crypt methods for streams and strings.
	switch {
	case v <= 3:
		h.stmMethod, h.strMethod = cryptRC4, cryptRC4
	case v == 4:
		h.stmMethod = cfMethod(enc, "StmF")
		h.strMethod = cfMethod(enc, "StrF")
	case v == 5:
		h.stmMethod, h.strMethod = cryptAESV3, cryptAESV3
	default:
		return nil, fmt.Errorf("unsupported /V %d", v)
	}

	// Derive the file key.
	if v == 5 {
		key, err := computeKeyV5(o, u, stringBytes(enc, "OE"), stringBytes(enc, "UE"))
		if err != nil {
			return nil, err
		}
		h.key = key
		return h, nil
	}

	keyLen := length / 8
	if v <= 1 {
		keyLen = 5
	}
	key, ok := computeKeyEmptyPassword(o, p, id, r, keyLen, encMeta, u)
	if !ok {
		return nil, ErrEncryptedNeedsPassword
	}
	h.key = key
	return h, nil
}

// Decrypt decrypts a string or stream body belonging to object (num, gen).
func (h *StdSecurityHandler) Decrypt(data []byte, num, gen int, isString bool) ([]byte, error) {
	method := h.stmMethod
	if isString {
		method = h.strMethod
	}
	switch method {
	case cryptIdentity:
		return data, nil
	case cryptRC4:
		return rc4Crypt(h.objectKey(num, gen, false), data), nil
	case cryptAESV2:
		return aesCBCDecrypt(h.objectKey(num, gen, true), data)
	case cryptAESV3:
		return aesCBCDecrypt(h.key, data)
	default:
		return data, nil
	}
}

// objectKey derives the per-object key for RC4/AESV2 (ISO 32000-1, 7.6.2,
// Algorithm 1): MD5(fileKey ++ objNum[3 LE] ++ gen[2 LE] ++ "sAlT" for AES).
func (h *StdSecurityHandler) objectKey(num, gen int, aesV2 bool) []byte {
	m := md5.New()
	m.Write(h.key)
	m.Write([]byte{byte(num), byte(num >> 8), byte(num >> 16)})
	m.Write([]byte{byte(gen), byte(gen >> 8)})
	if aesV2 {
		m.Write([]byte{0x73, 0x41, 0x6C, 0x54}) // "sAlT"
	}
	sum := m.Sum(nil)
	n := len(h.key) + 5
	if n > 16 {
		n = 16
	}
	return sum[:n]
}

// --- key derivation: RC4 / AESV2 (R2–R4) ---

// computeKeyEmptyPassword derives the file key for the empty password, trying the
// user password first and the owner password as a fallback.
func computeKeyEmptyPassword(o []byte, p int32, id []byte, r, keyLen int, encMeta bool, u []byte) ([]byte, bool) {
	// Empty user password.
	key := fileKeyRC4(nil, o, p, id, r, keyLen, encMeta)
	if validateUserPassword(key, u, id, r) {
		return key, true
	}
	// Empty owner password: recover the user password from /O, then re-derive.
	if userPw, ok := userPasswordFromOwner(nil, o, r, keyLen); ok {
		key = fileKeyRC4(userPw, o, p, id, r, keyLen, encMeta)
		if validateUserPassword(key, u, id, r) {
			return key, true
		}
	}
	return nil, false
}

// fileKeyRC4 implements Algorithm 2 (compute encryption key) for R2–R4.
func fileKeyRC4(pw, o []byte, p int32, id []byte, r, keyLen int, encMeta bool) []byte {
	h := md5.New()
	h.Write(padPassword(pw))
	if len(o) >= 32 {
		h.Write(o[:32])
	} else {
		h.Write(o)
	}
	up := uint32(p)
	h.Write([]byte{byte(up), byte(up >> 8), byte(up >> 16), byte(up >> 24)})
	h.Write(id)
	if r >= 4 && !encMeta {
		h.Write([]byte{0xff, 0xff, 0xff, 0xff})
	}
	key := h.Sum(nil)
	if r >= 3 {
		for i := 0; i < 50; i++ {
			s := md5.Sum(key[:keyLen])
			key = s[:]
		}
	}
	if keyLen > len(key) {
		keyLen = len(key)
	}
	return append([]byte(nil), key[:keyLen]...)
}

// validateUserPassword implements Algorithm 6 (R3+) / Algorithm 4 (R2).
func validateUserPassword(key, u, id []byte, r int) bool {
	if len(u) == 0 {
		return true // no /U to check against; accept the derived key
	}
	if r == 2 {
		want := rc4Crypt(key, padPassword(nil))
		return bytes.Equal(want, u[:min(len(u), 32)])
	}
	// R3+: MD5(padding ++ id), then 20 RC4 passes; compare first 16 bytes.
	m := md5.New()
	m.Write(passwordPad)
	m.Write(id)
	val := m.Sum(nil) // 16 bytes
	val = rc4Crypt(key, val)
	for i := 1; i <= 19; i++ {
		val = rc4Crypt(xorKey(key, byte(i)), val)
	}
	return bytes.Equal(val[:16], u[:min(len(u), 16)])
}

// userPasswordFromOwner implements Algorithm 7 (authenticate owner password):
// derive the RC4 owner key from the (empty) owner password and decrypt /O to
// recover the user password.
func userPasswordFromOwner(ownerPw, o []byte, r, keyLen int) ([]byte, bool) {
	if len(o) < 32 {
		return nil, false
	}
	h := md5.New()
	h.Write(padPassword(ownerPw))
	key := h.Sum(nil)
	if r >= 3 {
		for i := 0; i < 50; i++ {
			s := md5.Sum(key[:keyLen])
			key = s[:]
		}
	}
	if keyLen > len(key) {
		keyLen = len(key)
	}
	ownerKey := key[:keyLen]

	val := append([]byte(nil), o[:32]...)
	if r == 2 {
		val = rc4Crypt(ownerKey, val)
	} else {
		for i := 19; i >= 0; i-- {
			val = rc4Crypt(xorKey(ownerKey, byte(i)), val)
		}
	}
	return val, true
}

// --- key derivation: AESV3 (R6 / V5) ---

// computeKeyV5 derives the 32-byte AES-256 file key for the empty password
// (ISO 32000-2, Algorithm 2.A), trying the user then the owner path.
func computeKeyV5(o, u, oe, ue []byte) ([]byte, error) {
	if len(u) >= 48 && len(ue) >= 32 {
		valSalt, keySalt := u[32:40], u[40:48]
		if bytes.Equal(hash2B(nil, valSalt, nil), u[:32]) {
			ik := hash2B(nil, keySalt, nil)
			key, err := aesCBCNoPad(ik, make([]byte, 16), ue[:32])
			if err == nil {
				return key, nil
			}
		}
	}
	if len(o) >= 48 && len(oe) >= 32 && len(u) >= 48 {
		valSalt, keySalt := o[32:40], o[40:48]
		if bytes.Equal(hash2B(nil, valSalt, u[:48]), o[:32]) {
			ik := hash2B(nil, keySalt, u[:48])
			key, err := aesCBCNoPad(ik, make([]byte, 16), oe[:32])
			if err == nil {
				return key, nil
			}
		}
	}
	return nil, ErrEncryptedNeedsPassword
}

// hash2B implements the R6 hashing algorithm (ISO 32000-2, Algorithm 2.B).
func hash2B(pw, salt, udata []byte) []byte {
	h := sha256.New()
	h.Write(pw)
	h.Write(salt)
	h.Write(udata)
	k := h.Sum(nil)

	for round := 1; ; round++ {
		block := make([]byte, 0, len(pw)+len(k)+len(udata))
		block = append(block, pw...)
		block = append(block, k...)
		block = append(block, udata...)
		k1 := bytes.Repeat(block, 64)

		e, err := aesCBCEncryptNoPad(k[:16], k[16:32], k1)
		if err != nil {
			return k[:32]
		}
		var sum int
		for _, b := range e[:16] {
			sum += int(b)
		}
		switch sum % 3 {
		case 0:
			s := sha256.Sum256(e)
			k = s[:]
		case 1:
			s := sha512.Sum384(e)
			k = s[:]
		case 2:
			s := sha512.Sum512(e)
			k = s[:]
		}
		if round >= 64 && int(e[len(e)-1]) <= round-32 {
			break
		}
	}
	return k[:32]
}

// --- primitives ---

func padPassword(pw []byte) []byte {
	out := make([]byte, 32)
	n := copy(out, pw)
	copy(out[n:], passwordPad[:32-n])
	return out
}

func xorKey(key []byte, x byte) []byte {
	out := make([]byte, len(key))
	for i, b := range key {
		out[i] = b ^ x
	}
	return out
}

func rc4Crypt(key, data []byte) []byte {
	c, err := rc4.NewCipher(key)
	if err != nil {
		return data
	}
	out := make([]byte, len(data))
	c.XORKeyStream(out, data)
	return out
}

// aesCBCDecrypt decrypts CBC data whose first 16 bytes are the IV, stripping
// PKCS#7 padding.
func aesCBCDecrypt(key, data []byte) ([]byte, error) {
	if len(data) < aes.BlockSize {
		return nil, fmt.Errorf("aes: data shorter than IV")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	iv, ct := data[:aes.BlockSize], data[aes.BlockSize:]
	if len(ct)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("aes: ciphertext not block-aligned")
	}
	out := make([]byte, len(ct))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, ct)
	if n := len(out); n > 0 {
		if pad := int(out[n-1]); pad > 0 && pad <= aes.BlockSize && pad <= n {
			out = out[:n-pad]
		}
	}
	return out, nil
}

// aesCBCNoPad decrypts CBC data with an explicit IV and no padding (used for the
// V5 UE/OE intermediate-key step).
func aesCBCNoPad(key, iv, ct []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(ct)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("aes: ciphertext not block-aligned")
	}
	out := make([]byte, len(ct))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, ct)
	return out, nil
}

func aesCBCEncryptNoPad(key, iv, pt []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(pt)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("aes: plaintext not block-aligned")
	}
	out := make([]byte, len(pt))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, pt)
	return out, nil
}

// cfMethod resolves the crypt-filter method for the /StmF or /StrF entry (V4).
func cfMethod(enc Dict, which string) cryptMethod {
	name, _ := enc.Get(which).(Name)
	if name == "" || string(name) == "Identity" {
		return cryptIdentity
	}
	cf, _ := enc.Get("CF").(Dict)
	if cf == nil {
		return cryptIdentity
	}
	filter, _ := cf.Get(string(name)).(Dict)
	if filter == nil {
		return cryptIdentity
	}
	switch cfm, _ := filter.Get("CFM").(Name); string(cfm) {
	case "V2":
		return cryptRC4
	case "AESV2":
		return cryptAESV2
	case "AESV3":
		return cryptAESV3
	default:
		return cryptIdentity
	}
}

func intDefault(d Dict, key string, def int) int {
	if n, ok := d.Get(key).(Int); ok {
		return int(n)
	}
	return def
}

func boolDefault(d Dict, key string, def bool) bool {
	if b, ok := d.Get(key).(Bool); ok {
		return bool(b)
	}
	return def
}

func stringBytes(d Dict, key string) []byte {
	if s, ok := d.Get(key).(String); ok {
		return []byte(s)
	}
	return nil
}
