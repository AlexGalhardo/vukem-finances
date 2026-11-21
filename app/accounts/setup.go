package accounts

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"strings"
	"sync"
)

// SetupSecret guards /setup in the personal edition, so the first owner is
// not whoever reaches the port first. The secret is SETUP_TOKEN or, when that
// is unset, a one-time code generated at boot and printed in the boot log. It
// lives in memory only: a restart before setup prints a new code.
type SetupSecret struct {
	mu   sync.Mutex
	hash [sha256.Size]byte
	set  bool
	// Code is the generated code to print; empty when SETUP_TOKEN is used.
	Code string
}

// NewSetupSecret uses token (SETUP_TOKEN) when it is not empty, otherwise it
// generates a code with 128 bits or more of entropy.
func NewSetupSecret(token string) *SetupSecret {
	token = strings.TrimSpace(token)

	s := &SetupSecret{}
	if token == "" {
		token = rand.Text()
		s.Code = token
	}

	s.hash, s.set = sha256.Sum256([]byte(token)), true

	return s
}

// Check reports, in constant time, whether code is the secret. A spent
// secret matches nothing.
func (s *SetupSecret) Check(code string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	got := sha256.Sum256([]byte(strings.TrimSpace(code)))

	return subtle.ConstantTimeCompare(got[:], s.hash[:]) == 1 && s.set
}

// Spend retires the secret: the owner exists, setup is over.
func (s *SetupSecret) Spend() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.set = false
	s.hash = [sha256.Size]byte{}
}
