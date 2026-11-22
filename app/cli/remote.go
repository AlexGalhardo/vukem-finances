package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
	"unicode"

	"github.com/AlexGalhardo/vukem-finances/app/api"
	"github.com/AlexGalhardo/vukem-finances/app/operations"
)

// maxAnswer caps what a JSON answer may weigh before it is read whole.
const maxAnswer = 32 << 20

// NewHTTPClient is the client of remote mode: it never follows a redirect (a
// bearer token must go only where the person pointed it) and has a ceiling on
// the whole call.
func NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout:       5 * time.Minute,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// ParseServer checks the address of a server. A token travels in every call,
// so it must be https; plain http is accepted only for a loopback address.
func ParseServer(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return nil, usagef("the server %q is not a URL (https://host)", raw)
	}

	if u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, usagef("the server must be a plain URL, with no user, query or fragment")
	}

	switch u.Scheme {
	case "https":
	case "http":
		if !loopback(u.Hostname()) {
			return nil, usagef("the server must be https: a token is never sent over plain http (only to localhost)")
		}
	default:
		return nil, usagef("the server must be an https URL")
	}

	u.Path = strings.TrimRight(u.Path, "/")

	return u, nil
}

func loopback(host string) bool {
	if host == "localhost" {
		return true
	}

	ip := net.ParseIP(host)

	return ip != nil && ip.IsLoopback()
}

// Remote runs operations against the REST API of a server with a token.
type Remote struct {
	base  *url.URL
	token string
	hc    *http.Client
}

// NewRemote makes a remote backend. The token is kept only here and scrubbed
// from every error.
func NewRemote(server *url.URL, token string, hc *http.Client) *Remote {
	if hc == nil {
		hc = NewHTTPClient()
	}

	return &Remote{base: server, token: token, hc: hc}
}

// Close implements Backend.
func (r *Remote) Close() error { return nil }

// scrub keeps the token out of any text that is shown.
func (r *Remote) scrub(s string) string {
	if r.token == "" {
		return s
	}

	return strings.ReplaceAll(s, r.token, "[redacted]")
}

var downloadType = reflect.TypeFor[operations.Download]()

// Call implements Backend.
func (r *Remote) Call(ctx context.Context, rq Request) (Response, error) {
	cmd := rq.Cmd
	if cmd.Method == "" || cmd.Path == "" {
		return Response{}, fmt.Errorf("cli: %s has no REST route", cmd.Op.Name)
	}

	rest := map[string]any{}
	for k, v := range rq.Body {
		rest[k] = v
	}

	path := cmd.Path

	for _, p := range cmd.Params {
		v, ok := rest[p]
		if !ok {
			return Response{}, usagef("%s needs <%s>", cmd.Name(), p)
		}

		// PathEscape leaves dot segments alone, and they would reach another route.
		s := fmt.Sprint(v)
		if s == "." || s == ".." {
			return Response{}, usagef("<%s> cannot be %q", p, s)
		}

		path = strings.Replace(path, "{"+p+"}", url.PathEscape(s), 1)

		delete(rest, p)
	}

	escaped := r.base.EscapedPath() + api.BasePath + path

	decoded, err := url.PathUnescape(escaped)
	if err != nil {
		return Response{}, usagef("bad path: %v", err)
	}

	u := *r.base
	u.Path, u.RawPath = decoded, escaped

	var payload io.Reader

	if cmd.Method == http.MethodGet || cmd.Method == http.MethodDelete {
		u.RawQuery = query(rest).Encode()
	} else if len(rest) > 0 {
		raw, err := json.Marshal(rest)
		if err != nil {
			return Response{}, err
		}

		payload = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, cmd.Method, u.String(), payload)
	if err != nil {
		return Response{}, usagef("bad request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Accept", "application/json, */*;q=0.1")
	req.Header.Set("User-Agent", "vukem-finances-cli")

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if rq.IfMatch != "" {
		req.Header.Set("If-Match", rq.IfMatch)
	}

	if rq.Key != "" && cmd.Op.Idempotent {
		req.Header.Set("Idempotency-Key", rq.Key)
	}

	res, err := r.hc.Do(req)
	if err != nil {
		return Response{}, &Failure{Code: ExitFailure, Msg: r.scrub("cannot reach the server: " + urlError(err))}
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		defer res.Body.Close()

		return Response{}, r.problem(res)
	}

	if cmd.Op.Output == downloadType {
		return Response{Filename: filename(res), Stream: func(w io.Writer) error {
			defer res.Body.Close()

			_, err := io.Copy(w, res.Body)

			return err
		}}, nil
	}

	defer res.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(res.Body, maxAnswer+1))
	if err != nil || len(raw) > maxAnswer {
		return Response{}, &Failure{Code: ExitFailure, Msg: "the answer of the server is too large or was cut"}
	}

	if len(bytes.TrimSpace(raw)) == 0 {
		return Response{Value: map[string]any{}}, nil
	}

	v, err := decodeValue(raw)
	if err != nil {
		return Response{}, &Failure{Code: ExitFailure, Msg: "the server did not answer with JSON: is this a vukem-finances server?"}
	}

	// A link is an address of the REST surface; the operation's own answer has none.
	if m, ok := v.(map[string]any); ok {
		delete(m, "links")
	}

	return Response{Value: v}, nil
}

func urlError(err error) string {
	if ue, ok := errors.AsType[*url.Error](err); ok {
		return ue.Err.Error()
	}

	return err.Error()
}

func filename(res *http.Response) string {
	_, params, err := mime.ParseMediaType(res.Header.Get("Content-Disposition"))
	if err != nil {
		return ""
	}

	return params["filename"]
}

func query(in map[string]any) url.Values {
	q := url.Values{}

	for k, v := range in {
		switch x := v.(type) {
		case []string:
			for _, s := range x {
				q.Add(k, s)
			}
		case []any:
			for _, s := range x {
				q.Add(k, fmt.Sprint(s))
			}
		default:
			q.Set(k, fmt.Sprint(v))
		}
	}

	return q
}

// problem turns an error answer (problem+json) into a Failure.
func (r *Remote) problem(res *http.Response) error {
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 64<<10))

	var p struct {
		Code   string              `json:"code"`
		Detail string              `json:"detail"`
		Title  string              `json:"title"`
		Errors map[string][]string `json:"errors"`
	}

	_ = json.Unmarshal(raw, &p)

	if res.StatusCode >= 300 && res.StatusCode < 400 {
		return &Failure{Code: ExitFailure, Msg: "the server redirected the call: use its final address, the CLI never follows redirects with a token"}
	}

	code := p.Code
	if code == "" {
		code = api.CodeOf(res.StatusCode)
	}

	msg := p.Detail
	if msg == "" {
		msg = p.Title
	}

	if msg == "" {
		msg = http.StatusText(res.StatusCode)
	}

	msg += fieldProblems(p.Errors)

	if code != "" {
		msg = code + ": " + msg
	}

	return &Failure{Code: remoteExit(code, res.StatusCode), Msg: r.scrub(cleanText(msg))}
}

func remoteExit(code string, status int) int {
	switch code {
	case api.CodeUnauthorized:
		return ExitAuth
	case api.CodeBadRequest:
		return ExitUsage
	case api.CodeUnavailable:
		return ExitFailure
	}

	if c := ExitForCode(code); c != ExitFailure {
		return c
	}

	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return ExitAuth
	case status == http.StatusNotFound:
		return ExitNotFound
	case status == http.StatusConflict:
		return ExitConflict
	case status == http.StatusTooManyRequests || status == http.StatusPaymentRequired:
		return ExitLimit
	case status == http.StatusUnprocessableEntity || status == http.StatusBadRequest:
		return ExitUsage
	}

	return ExitFailure
}

// cleanText drops control characters, so text a server sends cannot move the
// cursor or rewrite the screen of the terminal.
func cleanText(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}

		return r
	}, s)
}
