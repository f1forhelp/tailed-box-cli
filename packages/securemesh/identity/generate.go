package identity

import (
	"crypto/ecdh"
	"crypto/ed25519"
	crand "crypto/rand"
	"errors"
	"strings"
	"time"

	securecrypto "github.com/f1forhelp/tailed-box-cli/packages/securemesh/crypto"
)

const (
	IdentityVersion = 1
	NetworkVersion  = 1
	NetworkIDBytes  = 32
)

var ErrInvalidGenerationInput = errors.New("invalid identity generation input")

func GenerateNetwork(createdAt time.Time, createdBy NodeID) (Network, error) {
	networkID, err := randomNetworkID()
	if err != nil {
		return Network{}, err
	}

	network := Network{
		Version:   NetworkVersion,
		ID:        networkID,
		CreatedAt: normalizeTime(createdAt),
		CreatedBy: createdBy,
	}
	if err := network.Validate(); err != nil {
		return Network{}, err
	}
	return network, nil
}

func GenerateIdentity(networkID NetworkID, role Role, createdAt time.Time) (Identity, error) {
	if err := networkID.Validate(); err != nil {
		return Identity{}, err
	}
	if err := role.Validate(); err != nil {
		return Identity{}, err
	}

	edPublic, edPrivate, err := ed25519.GenerateKey(crand.Reader)
	if err != nil {
		return Identity{}, err
	}
	xPrivate, err := ecdh.X25519().GenerateKey(crand.Reader)
	if err != nil {
		return Identity{}, err
	}
	xPublic := xPrivate.PublicKey()

	publicKeys := PublicKeySet{
		Signing: PublicKey{
			Algorithm: KeyAlgorithmEd25519,
			Bytes:     []byte(edPublic),
		},
		Transport: PublicKey{
			Algorithm: KeyAlgorithmX25519,
			Bytes:     xPublic.Bytes(),
		},
	}
	privateKeys := PrivateKeySet{
		Signing: PrivateKey{
			Algorithm: KeyAlgorithmEd25519,
			Bytes:     []byte(edPrivate),
		},
		Transport: PrivateKey{
			Algorithm: KeyAlgorithmX25519,
			Bytes:     xPrivate.Bytes(),
		},
	}

	nodeID, err := DeriveNodeID(publicKeys)
	if err != nil {
		return Identity{}, err
	}

	identity := Identity{
		Version:     IdentityVersion,
		NodeID:      nodeID,
		NetworkID:   networkID,
		Role:        role,
		PublicKeys:  publicKeys,
		PrivateKeys: privateKeys,
		CreatedAt:   normalizeTime(createdAt),
	}
	if err := identity.Validate(); err != nil {
		return Identity{}, err
	}
	return identity, nil
}

func DeriveNodeID(keys PublicKeySet) (NodeID, error) {
	if err := keys.Validate(); err != nil {
		return "", err
	}

	material := make([]byte, 0, 64+len(keys.Signing.Bytes)+len(keys.Transport.Bytes))
	material = append(material, []byte("tailed-box-cli node id v1")...)
	material = append(material, 0)
	material = append(material, []byte(keys.Signing.Algorithm)...)
	material = append(material, 0)
	material = append(material, keys.Signing.Bytes...)
	material = append(material, 0)
	material = append(material, []byte(keys.Transport.Algorithm)...)
	material = append(material, 0)
	material = append(material, keys.Transport.Bytes...)

	digest := securecrypto.SHA256(material)
	return NodeID("node_" + strings.ToLower(securecrypto.Base32NoPaddingEncode(digest))), nil
}

func randomNetworkID() (NetworkID, error) {
	random, err := securecrypto.RandomBytes(NetworkIDBytes)
	if err != nil {
		return "", err
	}
	return NetworkID("net_" + strings.ToLower(securecrypto.Base32NoPaddingEncode(random))), nil
}

func normalizeTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t.UTC()
}
