package tokens

import (
	"bytes"
	"encoding/base64"
	"math/big"
	"testing"
)

// Tests for encodeSignature/decodeSignature

func TestEncodeSignature_LeadingZeros(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		rBytes int
		sBytes int
	}{
		{"both 32 bytes", 32, 32},
		{"r has 1 leading zero", 31, 32},
		{"s has 1 leading zero", 32, 31},
		{"both have leading zeros", 31, 31},
		{"r has 2 leading zeros", 30, 32},
		{"s has 2 leading zeros", 32, 30},
		{"extreme: r is 1 byte", 1, 32},
		{"extreme: s is 1 byte", 32, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := new(big.Int).SetBytes(bytes.Repeat([]byte{0xFF}, tt.rBytes))
			s := new(big.Int).SetBytes(bytes.Repeat([]byte{0xAB}, tt.sBytes))

			encoded, err := encodeSignature(r, s)
			if err != nil {
				t.Fatalf("encodeSignature failed: %v", err)
			}

			decoded, err := base64.RawURLEncoding.DecodeString(encoded)
			if err != nil {
				t.Fatalf("base64 decode failed: %v", err)
			}

			if len(decoded) != 64 {
				t.Errorf("length = %d, want 64", len(decoded))
			}

			// Verify round-trip through decodeSignature
			rDec, sDec, err := decodeSignature(decoded)
			if err != nil {
				t.Fatalf("decodeSignature failed: %v", err)
			}

			if r.Cmp(rDec) != 0 {
				t.Errorf("r mismatch: got %x, want %x", rDec.Bytes(), r.Bytes())
			}
			if s.Cmp(sDec) != 0 {
				t.Errorf("s mismatch: got %x, want %x", sDec.Bytes(), s.Bytes())
			}
		})
	}
}

func TestDecodeSignature_InvalidLength(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		length int
	}{
		{"too short", 63},
		{"too long", 65},
		{"empty", 0},
		{"way too short", 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := decodeSignature(make([]byte, tt.length))
			if err == nil {
				t.Error("expected error for invalid length")
			}
		})
	}
}

// Tests for JWT section encoding/decoding

func TestEncodeDecodeJWTSection_RoundTrip(t *testing.T) {
	t.Parallel()
	type testStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	original := testStruct{Name: "test", Value: 42}
	encoded, err := encodeJWTSection(original)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded testStruct
	if err := decodeJWTSection(encoded, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded != original {
		t.Errorf("got %+v, want %+v", decoded, original)
	}
}

func TestDecodeJWTSection_InvalidBase64(t *testing.T) {
	t.Parallel()
	var result struct{}
	err := decodeJWTSection("not-valid-base64!!!", &result)
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestDecodeJWTSection_InvalidJSON(t *testing.T) {
	t.Parallel()
	encoded := base64.RawURLEncoding.EncodeToString([]byte("not-json"))
	var result struct{}
	err := decodeJWTSection(encoded, &result)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// Tests for JWT structure validation

func TestValidateStructure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid 3 parts", "a.b.c", false},
		{"2 parts", "a.b", true},
		{"4 parts", "a.b.c.d", true},
		{"1 part", "abc", true},
		{"empty", "", true},
		{"just dots", "..", false}, // 3 parts, all empty
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := validateStructure(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("got err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateStructure_ReturnsParts(t *testing.T) {
	t.Parallel()
	header, claims, sig, err := validateStructure("header.claims.signature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if header != "header" {
		t.Errorf("header = %q, want %q", header, "header")
	}
	if claims != "claims" {
		t.Errorf("claims = %q, want %q", claims, "claims")
	}
	if sig != "signature" {
		t.Errorf("signature = %q, want %q", sig, "signature")
	}
}

// Tests for header verification

func TestVerifyHeader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		header  JWTHeader
		wantErr bool
	}{
		{"valid ES256", JWTHeader{Algorithm: "ES256", Type: "JWT"}, false},
		{"wrong algorithm RS256", JWTHeader{Algorithm: "RS256", Type: "JWT"}, true},
		{"wrong algorithm HS256", JWTHeader{Algorithm: "HS256", Type: "JWT"}, true},
		{"wrong type JWS", JWTHeader{Algorithm: "ES256", Type: "JWS"}, true},
		{"wrong type JWE", JWTHeader{Algorithm: "ES256", Type: "JWE"}, true},
		{"both wrong", JWTHeader{Algorithm: "HS256", Type: "JWE"}, true},
		{"empty fields", JWTHeader{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifyHeader(&tt.header)
			if (err != nil) != tt.wantErr {
				t.Errorf("got err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

// Tests for helper functions

func TestBuildMessage(t *testing.T) {
	t.Parallel()
	result := buildMessage("header", "claims")
	if result != "header.claims" {
		t.Errorf("got %q, want %q", result, "header.claims")
	}
}

func TestBuildMessage_Empty(t *testing.T) {
	t.Parallel()
	result := buildMessage("", "")
	if result != "." {
		t.Errorf("got %q, want %q", result, ".")
	}
}

func TestHashMessage(t *testing.T) {
	t.Parallel()
	hash := hashMessage("test")
	if len(hash) != 32 { // SHA-256 = 32 bytes
		t.Errorf("hash length = %d, want 32", len(hash))
	}
}

func TestHashMessage_Deterministic(t *testing.T) {
	t.Parallel()
	hash1 := hashMessage("test")
	hash2 := hashMessage("test")
	if !bytes.Equal(hash1, hash2) {
		t.Error("hash should be deterministic")
	}
}

func TestHashMessage_DifferentInputs(t *testing.T) {
	t.Parallel()
	hash1 := hashMessage("test1")
	hash2 := hashMessage("test2")
	if bytes.Equal(hash1, hash2) {
		t.Error("different inputs should produce different hashes")
	}
}

func TestGenerateCSRFCode(t *testing.T) {
	t.Parallel()
	code1, err := generateCSRFCode()
	if err != nil {
		t.Fatalf("generateCSRFCode failed: %v", err)
	}
	if len(code1) == 0 {
		t.Error("empty CSRF code")
	}

	// Should be unique
	code2, err := generateCSRFCode()
	if err != nil {
		t.Fatalf("generateCSRFCode failed: %v", err)
	}
	if code1 == code2 {
		t.Error("CSRF codes should be unique")
	}
}

func TestNewES256JWTHeader(t *testing.T) {
	t.Parallel()
	header := newES256JWTHeader()
	if header.Algorithm != "ES256" {
		t.Errorf("Algorithm = %s, want ES256", header.Algorithm)
	}
	if header.Type != "JWT" {
		t.Errorf("Type = %s, want JWT", header.Type)
	}
}
