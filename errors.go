package hkontroller

import "fmt"

type PairVerifyError struct {
	Step string
	err  error
}

func (e *PairVerifyError) Unwrap() error {
	return e.err
}

func (e *PairVerifyError) Error() string {
	return fmt.Sprintf("pair-verify error on step %s: %v", e.Step, e.err)
}

type PairSetupError struct {
	Step string
	err  error
}

func (p *PairSetupError) Error() string {
	return fmt.Sprintf("pair-setup error on step %s: %v", p.Step, p.err)
}

func (p *PairSetupError) Unwrap() error {
	return p.err
}

type TlvError struct {
	Code    byte
	Message string
}

func (t *TlvError) Error() string {
	return fmt.Sprintf("tlv error %x: %s", t.Code, t.Message)
}

// Error codes for TLV8 communication.
var (
	TlvErrorUnknown        = TlvError{0x1, "unknown"}
	TlvErrorAuthentication = TlvError{0x2, "setup code or signature verification failed"}
	TlvErrorBackoff        = TlvError{0x3,
		"client must look at the retry delay TLV item and wait that many seconds before retrying"}
	TlvErrorMaxPeers    = TlvError{0x4, "server cannot accept any more pairings"}
	TlvErrorMaxTries    = TlvError{0x5, "server reached its maximum number of authentication attempts"}
	TlvErrorUnavailable = TlvError{0x6, "server pairing method is unavailable"}
	TlvErrorBusy        = TlvError{0x7, "server is busy and cannot accept a pairing request at this time"}

	tlvErrors = []TlvError{
		TlvErrorUnknown, TlvErrorAuthentication,
		TlvErrorBackoff, TlvErrorMaxPeers,
		TlvErrorMaxTries, TlvErrorUnavailable, TlvErrorBusy,
	}
)

func TlvErrorFromCode(code byte) error {
	for _, e := range tlvErrors {
		if e.Code == code {
			return &e
		}
	}
	return &TlvErrorUnknown
}
