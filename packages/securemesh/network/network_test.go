package network

import (
	"context"
	"errors"
	"testing"

	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
)

func TestSessionMetadataValidation(t *testing.T) {
	metadata := SessionMetadata{ID: "sess_1", NetworkID: "net_1", LocalNodeID: "node_local", RemoteNodeID: "node_remote", RemoteRole: identity.RoleWorker, Protocol: ProtocolUDP, State: SessionStateEstablished}
	if err := metadata.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestTransportInterfaceCompile(t *testing.T) {
	var _ Transport = fakeTransport{}
	var _ Session = fakeSession{}
	var _ PeerAuthenticator = fakeAuthenticator{}
}

type fakeTransport struct{}

func (fakeTransport) Listen(context.Context, ListenOptions) error { return nil }
func (fakeTransport) Dial(context.Context, DialOptions) (Session, error) {
	return fakeSession{}, nil
}
func (fakeTransport) Close() error { return nil }

type fakeSession struct{}

func (fakeSession) Metadata() SessionMetadata                { return SessionMetadata{} }
func (fakeSession) Send(context.Context, Message) error      { return nil }
func (fakeSession) Receive(context.Context) (Message, error) { return Message{}, errors.New("empty") }
func (fakeSession) Close() error                             { return nil }

type fakeAuthenticator struct{}

func (fakeAuthenticator) AuthenticatePeer(context.Context, SessionMetadata) error { return nil }
