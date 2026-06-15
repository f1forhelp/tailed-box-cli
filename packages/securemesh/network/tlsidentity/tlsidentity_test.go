package tlsidentity

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"testing"
	"time"

	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/config"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/peer"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/revocation"
)

func TestNewCertificateMetadataRoundTrip(t *testing.T) {
	fixture := newFixture(t)
	certificate, err := NewCertificate(fixture.worker, fixture.now)
	if err != nil {
		t.Fatalf("NewCertificate: %v", err)
	}
	if len(certificate.Certificate) != 1 {
		t.Fatalf("certificate chain length = %d, want 1", len(certificate.Certificate))
	}
	metadata, err := MetadataFromCertificate(certificate.Leaf)
	if err != nil {
		t.Fatalf("MetadataFromCertificate: %v", err)
	}
	if metadata.NodeID != fixture.worker.NodeID {
		t.Fatalf("node id = %q, want %q", metadata.NodeID, fixture.worker.NodeID)
	}
	if metadata.NetworkID != fixture.network.ID {
		t.Fatalf("network id = %q, want %q", metadata.NetworkID, fixture.network.ID)
	}
	if metadata.Role != identity.RoleWorker {
		t.Fatalf("role = %q, want worker", metadata.Role)
	}
}

func TestVerifierAcceptsValidPeer(t *testing.T) {
	fixture := newFixture(t)
	fixture.addWorkerPeer(t, fixture.worker.PublicKeys)
	metadata, err := fixture.verifier(identity.RoleWorker).Verify(fixture.workerRawCert(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if metadata.NodeID != fixture.worker.NodeID {
		t.Fatalf("node id = %q, want %q", metadata.NodeID, fixture.worker.NodeID)
	}
}

func TestVerifierRejectsUnknownPeer(t *testing.T) {
	fixture := newFixture(t)
	_, err := fixture.verifier(identity.RoleWorker).Verify(fixture.workerRawCert(t))
	if !errors.Is(err, ErrUnknownPeer) {
		t.Fatalf("Verify err = %v, want ErrUnknownPeer", err)
	}
}

func TestVerifierRejectsWrongNetwork(t *testing.T) {
	fixture := newFixture(t)
	fixture.addWorkerPeer(t, fixture.worker.PublicKeys)
	verifier := fixture.verifier(identity.RoleWorker)
	verifier.Options.NetworkID = identity.NetworkID("net_other")
	_, err := verifier.Verify(fixture.workerRawCert(t))
	if !errors.Is(err, ErrWrongNetwork) {
		t.Fatalf("Verify err = %v, want ErrWrongNetwork", err)
	}
}

func TestVerifierRejectsWrongRole(t *testing.T) {
	fixture := newFixture(t)
	fixture.addWorkerPeer(t, fixture.worker.PublicKeys)
	_, err := fixture.verifier(identity.RoleMaster).Verify(fixture.workerRawCert(t))
	if !errors.Is(err, ErrWrongRole) {
		t.Fatalf("Verify err = %v, want ErrWrongRole", err)
	}
}

func TestVerifierRejectsPublicKeyMismatch(t *testing.T) {
	fixture := newFixture(t)
	other, err := identity.GenerateIdentity(fixture.network.ID, identity.RoleWorker, fixture.now)
	if err != nil {
		t.Fatalf("GenerateIdentity other: %v", err)
	}
	fixture.addWorkerPeer(t, other.PublicKeys)
	_, err = fixture.verifier(identity.RoleWorker).Verify(fixture.workerRawCert(t))
	if !errors.Is(err, ErrPublicKeyMismatch) {
		t.Fatalf("Verify err = %v, want ErrPublicKeyMismatch", err)
	}
}

func TestVerifierRejectsRevokedPeer(t *testing.T) {
	fixture := newFixture(t)
	fixture.addWorkerPeer(t, fixture.worker.PublicKeys)
	if _, err := fixture.revocations.Revoke(fixture.worker.NodeID, fixture.worker.Role, fixture.master.NodeID, "test revoke"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	_, err := fixture.verifier(identity.RoleWorker).Verify(fixture.workerRawCert(t))
	if !errors.Is(err, ErrRevokedPeer) {
		t.Fatalf("Verify err = %v, want ErrRevokedPeer", err)
	}
}

func TestNewTLSConfigUsesTLS13AndALPN(t *testing.T) {
	fixture := newFixture(t)
	config, err := NewTLSConfig(fixture.master, fixture.verifier(identity.RoleWorker), fixture.now)
	if err != nil {
		t.Fatalf("NewTLSConfig: %v", err)
	}
	if config.MinVersion != tls.VersionTLS13 {
		t.Fatalf("MinVersion = %d, want TLS 1.3", config.MinVersion)
	}
	if len(config.NextProtos) != 1 || config.NextProtos[0] != ALPN {
		t.Fatalf("NextProtos = %#v, want %q", config.NextProtos, ALPN)
	}
	if len(config.Certificates) != 1 {
		t.Fatalf("cert count = %d, want 1", len(config.Certificates))
	}
	if config.VerifyPeerCertificate == nil {
		t.Fatal("VerifyPeerCertificate is nil")
	}
}

func TestMetadataRejectsCertificateWithoutMetadata(t *testing.T) {
	_, err := MetadataFromCertificate(&x509.Certificate{})
	if !errors.Is(err, ErrMissingMetadata) {
		t.Fatalf("MetadataFromCertificate err = %v, want ErrMissingMetadata", err)
	}
}

type fixture struct {
	now         time.Time
	network     identity.Network
	master      identity.Identity
	worker      identity.Identity
	peers       peer.Store
	revocations revocation.Store
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	network, err := identity.GenerateNetwork(now, "")
	if err != nil {
		t.Fatalf("GenerateNetwork: %v", err)
	}
	master, err := identity.GenerateIdentity(network.ID, identity.RoleMaster, now)
	if err != nil {
		t.Fatalf("GenerateIdentity master: %v", err)
	}
	worker, err := identity.GenerateIdentity(network.ID, identity.RoleWorker, now)
	if err != nil {
		t.Fatalf("GenerateIdentity worker: %v", err)
	}
	paths, err := config.NewPaths(t.TempDir())
	if err != nil {
		t.Fatalf("NewPaths: %v", err)
	}
	return fixture{
		now:         now,
		network:     network,
		master:      master,
		worker:      worker,
		peers:       peer.NewStore(paths),
		revocations: revocation.NewStore(paths).WithClock(func() time.Time { return now }),
	}
}

func (f fixture) addWorkerPeer(t *testing.T, publicKeys identity.PublicKeySet) {
	t.Helper()
	record := peer.Record{
		NodeID:     f.worker.NodeID,
		NetworkID:  f.network.ID,
		Role:       identity.RoleWorker,
		PublicKeys: publicKeys,
		Status:     peer.StatusActive,
		AddedAt:    f.now,
	}
	if err := f.peers.Add(record); err != nil {
		t.Fatalf("Add peer: %v", err)
	}
}

func (f fixture) verifier(expectedRole identity.Role) Verifier {
	return Verifier{Options: VerifyOptions{
		NetworkID:    f.network.ID,
		ExpectedRole: expectedRole,
		Peers:        f.peers,
		Revocations:  f.revocations,
		Now:          func() time.Time { return f.now },
	}}
}

func (f fixture) workerRawCert(t *testing.T) [][]byte {
	t.Helper()
	certificate, err := NewCertificate(f.worker, f.now)
	if err != nil {
		t.Fatalf("NewCertificate worker: %v", err)
	}
	return certificate.Certificate
}
