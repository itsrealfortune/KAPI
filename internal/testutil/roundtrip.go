package testutil

import "net/http"

// RoundTripFunc is an http.RoundTripper backed by a plain function.
// Use it to stub http.DefaultTransport or an http.Client.Transport in tests:
//
//	http.DefaultTransport = testutil.RoundTripFunc(func(r *http.Request) (*http.Response, error) {
//	    // return a fake response
//	})
type RoundTripFunc func(*http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
