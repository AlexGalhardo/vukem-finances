package offline_test

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/textproto"
	"os"
	"syscall"
	"testing"

	"github.com/AlexGalhardo/vukem/mail"

	"github.com/AlexGalhardo/vukem-finances/app/offline"
)

func TestIsConnectivity(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"dns", fmt.Errorf("mail: smtp dial: %w", &net.DNSError{Err: "no such host", Name: "mailpit", IsNotFound: true}), true},
		{"refused", fmt.Errorf("mail: smtp dial: %w", &net.OpError{Op: "dial", Net: "tcp", Err: os.NewSyscallError("connect", syscall.ECONNREFUSED)}), true},
		{"a dial that failed for any reason", &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("network is unreachable")}, true},
		{"unreachable errno", fmt.Errorf("send: %w", syscall.ENETUNREACH), true},
		{"deadline", fmt.Errorf("send: %w", context.DeadlineExceeded), true},
		{"authentication", &textproto.Error{Code: 535, Msg: "authentication failed"}, false},
		{"recipient", &textproto.Error{Code: 550, Msg: "no such user"}, false},
		{"tls certificate", x509.UnknownAuthorityError{}, false},
		{"a connection reset while talking", &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET}, false},
		{"a plain error", errors.New("mail: no recipients"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := offline.IsConnectivity(tc.err); got != tc.want {
				t.Errorf("IsConnectivity(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

type failing struct{ err error }

func (f failing) Send(context.Context, *mail.Message) error { return f.err }

func TestBestEffort(t *testing.T) {
	t.Parallel()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	offlineErr := &net.DNSError{Err: "no such host", Name: "smtp.example.com"}
	authErr := &textproto.Error{Code: 535, Msg: "bad credentials"}

	cases := []struct {
		name    string
		err     error
		inline  bool
		wantErr error
	}{
		{"delivered", nil, true, nil},
		{"offline on an inline queue is dropped, so the answer is the same for everyone", offlineErr, true, nil},
		{"offline on a real queue is returned, so the queue retries it", offlineErr, false, offlineErr},
		{"authentication is returned on an inline queue", authErr, true, authErr},
		{"authentication is returned on a real queue", authErr, false, authErr},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			send := offline.BestEffort(failing{tc.err}, log, func() bool { return tc.inline })

			if err := send.Send(t.Context(), &mail.Message{}); !errors.Is(err, tc.wantErr) {
				t.Errorf("Send() = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestBestEffortLeavesTransportsThatNeverUseTheNetwork(t *testing.T) {
	t.Parallel()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	array := mail.NewArray()

	if got := offline.BestEffort(array, log, func() bool { return true }); got != mail.Transport(array) {
		t.Errorf("the array transport was wrapped: %T", got)
	}

	if got := offline.BestEffort(mail.Log{L: log}, log, func() bool { return true }); got != mail.Transport(mail.Log{L: log}) {
		t.Errorf("the log transport was wrapped: %T", got)
	}
}
