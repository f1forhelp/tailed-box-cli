package join

import (
	"errors"
	"fmt"
	"strings"
	"time"

	securecrypto "github.com/f1forhelp/tailed-box-cli/packages/securemesh/crypto"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
)

const (
	CodeSecretBytes       = 32
	VerifierSaltBytes     = 32
	VerifierAlgorithmV1   = "hmac-sha256-v1"
	RecordVersion         = 1
	codeIDDomainSeparator = "tailed-box-cli join code id v1"
)

var (
	ErrInvalidCode  = errors.New("invalid join code")
	ErrCodeConsumed = errors.New("join code already consumed")
	ErrWrongNetwork = errors.New("join code is for a different network")
	ErrWrongRole    = errors.New("join code is for a different role")
)

func GenerateCode() (string, error) {
	return securecrypto.RandomBase32(CodeSecretBytes)
}

func NewRecord(request CreateRequest, createdAt time.Time) (string, Record, error) {
	if err := request.Validate(); err != nil {
		return "", Record{}, err
	}

	code, err := GenerateCode()
	if err != nil {
		return "", Record{}, err
	}
	salt, err := securecrypto.RandomBytes(VerifierSaltBytes)
	if err != nil {
		return "", Record{}, err
	}
	verifier, err := VerifierForCode(code, salt)
	if err != nil {
		return "", Record{}, err
	}

	record := Record{
		ID:                CodeIDForVerifier(verifier),
		NetworkID:         request.NetworkID,
		ExpectedRole:      request.ExpectedRole,
		IssuedBy:          request.IssuedBy,
		CreatedAt:         normalizeTime(createdAt),
		VerifierAlgorithm: VerifierAlgorithmV1,
		Salt:              salt,
		Verifier:          verifier,
		Status:            StatusUnused,
	}
	if err := record.Validate(); err != nil {
		return "", Record{}, err
	}
	return code, record, nil
}

func VerifierForCode(code string, salt []byte) ([]byte, error) {
	if strings.TrimSpace(code) == "" {
		return nil, ErrInvalidCode
	}
	if len(salt) == 0 {
		return nil, fmt.Errorf("%w: missing verifier salt", ErrInvalidRecord)
	}
	secret, err := securecrypto.Base32NoPaddingDecode(code)
	if err != nil {
		return nil, fmt.Errorf("%w: malformed code", ErrInvalidCode)
	}
	return securecrypto.HMACSHA256(salt, secret), nil
}

func CodeIDForVerifier(verifier []byte) CodeID {
	material := make([]byte, 0, len(codeIDDomainSeparator)+1+len(verifier))
	material = append(material, []byte(codeIDDomainSeparator)...)
	material = append(material, 0)
	material = append(material, verifier...)
	digest := securecrypto.SHA256(material)
	return CodeID("join_" + strings.ToLower(securecrypto.Base32NoPaddingEncode(digest)))
}

func (r CreateRequest) Validate() error {
	if err := r.NetworkID.Validate(); err != nil {
		return err
	}
	if err := r.ExpectedRole.Validate(); err != nil {
		return err
	}
	if err := r.IssuedBy.Validate(); err != nil {
		return err
	}
	return nil
}

func (r ConsumeRequest) Validate() error {
	if strings.TrimSpace(r.Code) == "" {
		return ErrInvalidCode
	}
	if err := r.NetworkID.Validate(); err != nil {
		return err
	}
	if err := r.ExpectedRole.Validate(); err != nil {
		return err
	}
	if err := r.ConsumedBy.Validate(); err != nil {
		return err
	}
	return nil
}

func normalizeTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t.UTC()
}

func sameNetwork(a, b identity.NetworkID) bool {
	return a == b
}
