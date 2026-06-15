package join

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/config"
	securecrypto "github.com/f1forhelp/tailed-box-cli/packages/securemesh/crypto"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
)

func TestGenerateCodeHighEntropyExpectedLength(t *testing.T) {
	first, err := GenerateCode()
	if err != nil {
		t.Fatalf("GenerateCode first: %v", err)
	}
	second, err := GenerateCode()
	if err != nil {
		t.Fatalf("GenerateCode second: %v", err)
	}
	if first == second {
		t.Fatal("two generated join codes matched")
	}
	if len(first) != 52 {
		t.Fatalf("code length = %d, want 52", len(first))
	}
	decoded, err := securecrypto.Base32NoPaddingDecode(first)
	if err != nil {
		t.Fatalf("decode generated code: %v", err)
	}
	if len(decoded) != CodeSecretBytes {
		t.Fatalf("decoded length = %d, want %d", len(decoded), CodeSecretBytes)
	}
}

func TestNewRecordDoesNotPersistPlaintextCode(t *testing.T) {
	code, record, err := NewRecord(validCreateRequest(), time.Now())
	if err != nil {
		t.Fatalf("NewRecord: %v", err)
	}
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(data), code) {
		t.Fatal("record JSON contains plaintext join code")
	}
	if record.Status != StatusUnused || record.ConsumedAt != nil {
		t.Fatalf("new record consumed state = %#v", record)
	}
}

func TestStoreSingleUseBehavior(t *testing.T) {
	store := newTestStore(t, time.Now)
	code, record, err := store.Create(validCreateRequest())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	result, err := store.ValidateAndConsume(ConsumeRequest{
		Code:         code,
		NetworkID:    record.NetworkID,
		ExpectedRole: record.ExpectedRole,
		ConsumedBy:   identity.NodeID("node_consumer"),
	})
	if err != nil {
		t.Fatalf("ValidateAndConsume: %v", err)
	}
	if !result.Consumed || result.Record.Status != StatusConsumed {
		t.Fatalf("consume result = %#v", result)
	}
	_, err = store.ValidateAndConsume(ConsumeRequest{
		Code:         code,
		NetworkID:    record.NetworkID,
		ExpectedRole: record.ExpectedRole,
		ConsumedBy:   identity.NodeID("node_consumer"),
	})
	if !errors.Is(err, ErrCodeConsumed) {
		t.Fatalf("second consume err = %v, want ErrCodeConsumed", err)
	}
}

func TestStoreRejectsInvalidWrongRoleAndWrongNetwork(t *testing.T) {
	store := newTestStore(t, time.Now)
	code, record, err := store.Create(validCreateRequest())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	_, err = store.ValidateAndConsume(ConsumeRequest{Code: "not-a-code", NetworkID: record.NetworkID, ExpectedRole: record.ExpectedRole, ConsumedBy: "node_consumer"})
	if !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("invalid code err = %v, want ErrInvalidCode", err)
	}
	_, err = store.ValidateAndConsume(ConsumeRequest{Code: code, NetworkID: record.NetworkID, ExpectedRole: identity.RoleMaster, ConsumedBy: "node_consumer"})
	if !errors.Is(err, ErrWrongRole) {
		t.Fatalf("wrong role err = %v, want ErrWrongRole", err)
	}
	_, err = store.ValidateAndConsume(ConsumeRequest{Code: code, NetworkID: identity.NetworkID("net_other"), ExpectedRole: record.ExpectedRole, ConsumedBy: "node_consumer"})
	if !errors.Is(err, ErrWrongNetwork) {
		t.Fatalf("wrong network err = %v, want ErrWrongNetwork", err)
	}
}

func TestNoMandatoryExpiry(t *testing.T) {
	old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	store := newTestStore(t, func() time.Time { return old })
	code, record, err := store.Create(validCreateRequest())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	store = store.WithClock(func() time.Time { return old.AddDate(5, 0, 0) })
	if _, err := store.ValidateAndConsume(ConsumeRequest{Code: code, NetworkID: record.NetworkID, ExpectedRole: record.ExpectedRole, ConsumedBy: "node_consumer"}); err != nil {
		t.Fatalf("old code should not expire in this milestone: %v", err)
	}
}

func TestStoreDoesNotPersistPlaintextCode(t *testing.T) {
	paths, err := config.NewPaths(t.TempDir())
	if err != nil {
		t.Fatalf("NewPaths: %v", err)
	}
	store := NewStore(paths)
	code, _, err := store.Create(validCreateRequest())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	data, err := os.ReadFile(paths.JoinCodesPath())
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(data), code) {
		t.Fatal("join-code state contains plaintext code")
	}
}

func newTestStore(t *testing.T, now func() time.Time) Store {
	t.Helper()
	paths, err := config.NewPaths(t.TempDir())
	if err != nil {
		t.Fatalf("NewPaths: %v", err)
	}
	return NewStore(paths).WithClock(now)
}

func validCreateRequest() CreateRequest {
	return CreateRequest{NetworkID: identity.NetworkID("net_test"), ExpectedRole: identity.RoleWorker, IssuedBy: identity.NodeID("node_master")}
}
