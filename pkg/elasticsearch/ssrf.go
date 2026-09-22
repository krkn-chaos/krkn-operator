/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package elasticsearch

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// ErrDestinationNotAllowed is returned when an inline (user-supplied)
// Elasticsearch destination violates the destination policy: a disallowed
// scheme or port, or a host that resolves to a loopback, private, link-local,
// carrier-grade-NAT, metadata, multicast, or unspecified address. It is a
// security boundary, not a formatting check, and callers use errors.Is to
// detect it.
var ErrDestinationNotAllowed = errors.New("destination not allowed")

// allowedInlinePorts is the fixed set of ports an inline connection may target.
// The inline path is reachable by any authenticated user, so the target port is
// constrained to the ports Elasticsearch/OpenSearch clusters are actually served
// on (443 for TLS front ends, 9200/9243 for the native REST API) rather than
// allowing an arbitrary port to be probed. Admin-created saved configs are not
// subject to this policy and may use any port.
var allowedInlinePorts = map[int]struct{}{
	443:  {},
	9200: {},
	9243: {},
}

// maxInlineRedirects bounds how many redirect hops an inline request may follow.
// Each hop is re-validated against the destination policy, but the count is also
// capped to prevent redirect loops from a permitted host being used to probe
// many internal targets in a single request.
const maxInlineRedirects = 3

// resolveTimeout bounds a single DNS resolution performed by the destination
// guard so a slow resolver cannot block a request indefinitely.
const resolveTimeout = 5 * time.Second

// cgnatBlock is the RFC 6598 carrier-grade-NAT range (100.64.0.0/10). Go's
// net.IP.IsPrivate does not classify it, but it is not a valid public
// destination and can front internal infrastructure, so it is rejected
// explicitly.
var cgnatBlock = &net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}

// validateDestinationIP rejects an address that must never be reachable from the
// operator's network position via a user-supplied inline connection. It blocks
// loopback, private (RFC1918 + IPv6 ULA), link-local unicast (which covers the
// 169.254.169.254 cloud/Kubernetes metadata service and IPv6 fe80::/10),
// link-local and general multicast, the unspecified address, and carrier-grade
// NAT. IPv4-in-IPv6 mapped addresses are normalized first so a mapped internal
// address cannot slip through.
func validateDestinationIP(ip net.IP) error {
	if ip == nil {
		return fmt.Errorf("%w: nil address", ErrDestinationNotAllowed)
	}
	// Normalize IPv4-mapped IPv6 (e.g. ::ffff:127.0.0.1) to its IPv4 form so the
	// classification checks below see the real address family.
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	switch {
	case ip.IsLoopback():
		return fmt.Errorf("%w: loopback address %s", ErrDestinationNotAllowed, ip)
	case ip.IsPrivate():
		return fmt.Errorf("%w: private address %s", ErrDestinationNotAllowed, ip)
	case ip.IsLinkLocalUnicast():
		return fmt.Errorf("%w: link-local address %s", ErrDestinationNotAllowed, ip)
	case ip.IsLinkLocalMulticast(), ip.IsMulticast():
		return fmt.Errorf("%w: multicast address %s", ErrDestinationNotAllowed, ip)
	case ip.IsUnspecified():
		return fmt.Errorf("%w: unspecified address %s", ErrDestinationNotAllowed, ip)
	case cgnatBlock.Contains(ip):
		return fmt.Errorf("%w: carrier-grade NAT address %s", ErrDestinationNotAllowed, ip)
	}
	return nil
}

// validateDestinationURL enforces the scheme and port policy on a target URL. It
// accepts https on any allowed port and http only on an allowed port (the
// separate plaintext-credential guard still forbids credentials over http). Any
// other scheme, a missing/invalid port, or a port outside the allowlist is
// rejected. It does not resolve DNS; see validateInlineDestination for the full
// pre-flight check.
func validateDestinationURL(u *url.URL) error {
	switch u.Scheme {
	case "https", "http":
	default:
		return fmt.Errorf("%w: scheme %q is not permitted", ErrDestinationNotAllowed, u.Scheme)
	}
	portStr := u.Port()
	if portStr == "" {
		return fmt.Errorf("%w: a port is required", ErrDestinationNotAllowed)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("%w: invalid port %q", ErrDestinationNotAllowed, portStr)
	}
	if _, ok := allowedInlinePorts[port]; !ok {
		return fmt.Errorf("%w: port %d is not permitted", ErrDestinationNotAllowed, port)
	}
	return nil
}

// validateInlineDestination is the pre-flight destination check for a restricted
// (inline) connection. It runs before any outbound request is made: it parses
// base, enforces the scheme/port policy, resolves the host, and rejects the
// request if any resolved address is disallowed. Resolving here (and dialing the
// exact validated address later, see guardedDialContext) means a name cannot
// pass this check and then be re-resolved to an internal address at connect time
// (DNS rebinding). It returns an error wrapping ErrDestinationNotAllowed on any
// violation.
func validateInlineDestination(ctx context.Context, base string) error {
	u, err := url.Parse(base)
	if err != nil {
		return fmt.Errorf("%w: invalid destination URL: %v", ErrDestinationNotAllowed, err)
	}
	if err := validateDestinationURL(u); err != nil {
		return err
	}

	host := u.Hostname()
	// A literal IP is validated directly; a hostname is resolved and every
	// returned address must pass, so a name with mixed public/internal records
	// cannot be used to reach an internal target.
	if ip := net.ParseIP(host); ip != nil {
		return validateDestinationIP(ip)
	}

	resolveCtx, cancel := context.WithTimeout(ctx, resolveTimeout)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(resolveCtx, host)
	if err != nil {
		return fmt.Errorf("%w: cannot resolve host %q: %v", ErrDestinationNotAllowed, host, err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("%w: host %q resolved to no addresses", ErrDestinationNotAllowed, host)
	}
	for _, a := range addrs {
		if err := validateDestinationIP(a.IP); err != nil {
			return err
		}
	}
	return nil
}

// guardedDialContext returns a net.Conn dialer that re-validates the address it
// is about to connect to and dials that exact resolved IP. Re-validating at dial
// time (in addition to the pre-flight validateInlineDestination) closes the
// DNS-rebinding window: even if a name re-resolves between the pre-flight check
// and the dial, the connection is only made to an address that passes the
// policy. The dialer resolves the host itself and connects to the validated IP
// literal so the kernel does not perform a second, unchecked resolution.
func guardedDialContext() func(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: queryTimeout}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid dial address %q: %v", ErrDestinationNotAllowed, addr, err)
		}
		if ip := net.ParseIP(host); ip != nil {
			if err := validateDestinationIP(ip); err != nil {
				return nil, err
			}
			return dialer.DialContext(ctx, network, addr)
		}
		resolveCtx, cancel := context.WithTimeout(ctx, resolveTimeout)
		defer cancel()
		addrs, err := net.DefaultResolver.LookupIPAddr(resolveCtx, host)
		if err != nil {
			return nil, fmt.Errorf("%w: cannot resolve host %q: %v", ErrDestinationNotAllowed, host, err)
		}
		var lastErr error
		for _, a := range addrs {
			if err := validateDestinationIP(a.IP); err != nil {
				lastErr = err
				continue
			}
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(a.IP.String(), port))
			if err != nil {
				lastErr = err
				continue
			}
			return conn, nil
		}
		if lastErr == nil {
			lastErr = fmt.Errorf("%w: host %q resolved to no usable address", ErrDestinationNotAllowed, host)
		}
		return nil, lastErr
	}
}

// guardedCheckRedirect enforces the destination policy on every redirect hop and
// caps the number of hops. Without it a permitted host could 30x-redirect the
// operator into an internal address, bypassing the pre-flight check. Each hop's
// scheme/port is validated; the dial-time guard (guardedDialContext) still
// validates the resolved address the redirected request connects to.
func guardedCheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxInlineRedirects {
		return fmt.Errorf("%w: too many redirects", ErrDestinationNotAllowed)
	}
	return validateDestinationURL(req.URL)
}
