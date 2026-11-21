package ai

import (
	"net/http"
	"net/netip"
	"net/url"
	"strings"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/httpclient"
)

func badURL(msg string) error { return errs.New(http.StatusUnprocessableEntity, msg) }

// CheckBaseURL validates the base URL of a credential at save time and again
// when a provider is built. Only openai_compatible takes one. It must be
// https. With allowPrivate (AI_ALLOW_PRIVATE_HOSTS, personal edition only) a
// local model server is also fine over http, but only when the host is
// localhost or a private IP literal: http to a name that could be public would
// send the key in clear. This is the early, readable refusal; the connect-time
// check of the safe client is the one that holds, after DNS resolution and on
// every redirect.
func CheckBaseURL(kind, raw string, allowPrivate bool) error {
	raw = strings.TrimSpace(raw)

	if kind != KindOpenAICompatible {
		if raw != "" {
			return badURL("A base URL is only used with the OpenAI-compatible provider.")
		}

		return nil
	}

	if raw == "" {
		return badURL("Enter the base URL of the server, such as https://api.example.com/v1.")
	}

	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.Hostname() == "" {
		return badURL("The base URL is not a valid URL.")
	}

	if u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return badURL("The base URL cannot hold a user, a query or a fragment.")
	}

	host := strings.ToLower(u.Hostname())
	local := isLocalHost(host)

	var literal netip.Addr
	if ip, err := netip.ParseAddr(host); err == nil {
		literal = ip
	}

	switch u.Scheme {
	case "https":
	case "http":
		if !allowPrivate || !local {
			return badURL("The base URL must use https.")
		}
	default:
		return badURL("The base URL must use https.")
	}

	if literal.IsValid() {
		var allow []netip.Prefix
		if allowPrivate {
			allow = localPrefixes
		}

		if !httpclient.Allowed(literal, allow) {
			return badURL("That address is not allowed.")
		}
	} else if (host == "localhost" || strings.HasSuffix(host, ".localhost")) && !allowPrivate {
		return badURL("That address is not allowed.")
	}

	return nil
}

// isLocalHost is true for localhost and for IP literals of this machine or a
// private network: the only hosts that may be reached over http.
func isLocalHost(host string) bool {
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}

	ip, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}

	ip = ip.Unmap()
	for _, p := range localPrefixes {
		if p.Contains(ip) {
			return true
		}
	}

	return false
}
