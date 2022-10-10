package hkontrol

import (
	"github.com/hkontrol/hkontroller/hkdf"
	"github.com/tadglines/go-pkgs/crypto/srp"

	"crypto/sha512"
)

// Main SRP algorithm is described in http://srp.stanford.edu/design.html
// The HAP uses the SRP-6a Stanford implementation with the following characteristics
//      x = H(s | H(I | ":" | P)) -> called the key derivative function
//      M1 = H(H(N) xor H(g), H(I), s, A, B, K)
const (
	srpGroup = "rfc5054.3072" // N (modulo) => 384 byte
)

type pairSetupClientSession struct {
	PublicKey     []byte   // A
	SessionKey    []byte   // K
	EncryptionKey [32]byte // K
	Salt          []byte   // s
	Proof         []byte

	session *srp.ClientSession
}

func (s *pairSetupClientSession) SetupEncryptionKey(salt []byte, info []byte) error {
	hash, err := hkdf.Sha512(s.SessionKey, salt, info)
	if err == nil {
		s.EncryptionKey = hash
	}

	return err
}

// newPairSetupSession return a new setup server session.
func newPairSetupClientSession(serverSalt []byte, serverB []byte, pin string) (*pairSetupClientSession, error) {
	var err error
	userName := []byte("Pair-Setup")

	srp_, err := srp.NewSRP(srpGroup, sha512.New, keyDerivativeFuncRFC2945(sha512.New, userName))

	if err == nil {
		srp_.SaltLength = 16

		if err == nil {
			srpClient := srp_.NewClientSession(userName, []byte(pin))
			key, err := srpClient.ComputeKey(serverSalt, serverB)
			if err != nil {
				return nil, err
			}

			pairing := pairSetupClientSession{
				session:    srpClient,
				PublicKey:  srpClient.GetA(),
				SessionKey: key,
				Proof:      srpClient.ComputeAuthenticator(),
			}
			return &pairing, nil
		}
	}

	return nil, err
}

// keyDerivativeFuncRFC2945 returns the SRP-6a key derivative function which does
//      x = H(s | H(I | ":" | P))
func keyDerivativeFuncRFC2945(h srp.HashFunc, id []byte) srp.KeyDerivationFunc {
	return func(salt, pin []byte) []byte {
		h := h()
		h.Write(id)
		h.Write([]byte(":"))
		h.Write(pin)
		t2 := h.Sum(nil)
		h.Reset()
		h.Write(salt)
		h.Write(t2)
		return h.Sum(nil)
	}
}
