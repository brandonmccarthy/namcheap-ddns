package main

import (
	"fmt"
	nc "github.com/billputer/go-namecheap"
	"net/http"
	"net/http/httptest"
	"testing"
)

type client struct {
	ApiUser  string
	ApiToken string
	UserName string
	BaseURL  string
}

func newClient(user, token, username string) *client {
	return &client{
		ApiUser:  user,
		ApiToken: token,
		UserName: username,
	}
}

func (client *client) DomainDNSSetHosts(sld, tld string, hosts []nc.DomainDNSHost) (*nc.DomainDNSSetHostsResult, error) {
	for _, host := range hosts {
		if host.Address == "127.0.0.1" {
			return nil, fmt.Errorf("setting host to localhost is forbidden")
		}
	}
	return &nc.DomainDNSSetHostsResult{
		Domain:    fmt.Sprintf("%s.%s", sld, tld),
		IsSuccess: true,
	}, nil
}

// Run a fake HTTP server that will shutdown once a GET request is fulfilled
func mockHTTPServer(label, input string, t *testing.T) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/"+label, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, input)
	})
	return mux
}

func TestGetLocalIP(t *testing.T) {
	for _, c := range []struct {
		label   string
		input   string
		want    string
		wantErr bool
	}{
		{
			label:   "success",
			input:   "192.168.10.1\n\n",
			want:    "192.168.10.1",
			wantErr: false,
		},
		{
			label:   "hardparse",
			input:   "Your IP address is: 192.168.30.1\n\n",
			want:    "192.168.30.1",
			wantErr: false,
		},
		{
			label:   "failure",
			input:   "EROROROORORO\n\n",
			want:    "nil",
			wantErr: true,
		},
	} {
		t.Run(c.label, func(t *testing.T) {
			srv := httptest.NewServer(mockHTTPServer(c.label, c.input, t))
			ip, err := getLocalIP(fmt.Sprintf("%s/%s", srv.URL, c.label))
			defer srv.Close()
			if err != nil && !c.wantErr {
				t.Errorf("Test %s failed: received error, did not want error: %v", c.label, err)
			}
			if err == nil && c.wantErr {
				t.Errorf("Test %s failed: did not receive error, wanted an error", c.label)
			}
			if err == nil && ip.String() != c.want {
				t.Errorf("Test %s failed: got %s, wanted %s", c.label, ip.String(), c.want)
			}
		})
	}
}

func TestUpdateDomain(t *testing.T) {
	d := newClient("test", "testing-token", "Testing McTester")
	for _, c := range []struct {
		label     string
		fqdn      string
		ipAddress string
		wantErr   bool
	}{
		{
			label:     "bad-localhost",
			fqdn:      "somehost.sld.tld",
			ipAddress: "127.0.0.1",
			wantErr:   true,
		},
		{
			label:     "success",
			fqdn:      "somehost.sld.tld",
			ipAddress: "123.13.31.30",
			wantErr:   false,
		},
	} {
		t.Run(c.label, func(t *testing.T) {
			err := updateDomain(d, c.fqdn, c.ipAddress)
			if err != nil && !c.wantErr {
				t.Errorf("Test %s failed: got error, didn't want error: %v", c.label, err)
			}
			if err == nil && c.wantErr {
				t.Errorf("Test %s failed: did not get an error, wanted an error.", c.label)
			}
		})
	}
}

func TestReverse(t *testing.T) {
	for _, c := range []struct {
		label string
		input []string
		want  []string
	}{
		{
			label: "number success",
			input: []string{"one", "two", "three"},
			want:  []string{"three", "two", "one"},
		},
		{
			label: "alpha success",
			input: []string{"abc", "def", "ghi", "jkl"},
			want:  []string{"jkl", "ghi", "def", "abc"},
		},
	} {
		t.Run(c.label, func(t *testing.T) {
			res := reverse(c.input)
			for k, v := range res {
				if c.want[k] != v {
					t.Errorf("Test failed: Got %v, want %v", res, c.want)
				}
			}
		})
	}
}

func TestParseFQDN(t *testing.T) {
	for _, c := range []struct {
		label   string
		input   string
		want    fqdn
		wantErr bool
	}{
		{
			label: "success",
			input: "somehost.corp.com",
			want: fqdn{
				TLD:       "com",
				SLD:       "corp",
				Subdomain: "somehost",
			},
		},
		{
			label:   "bad parse",
			input:   "google.com",
			wantErr: true,
		},
		{
			label: "long parse",
			input: "sub.subdomain.foo.bar",
			want: fqdn{
				TLD:       "bar",
				SLD:       "foo",
				Subdomain: "subdomain",
			},
		},
	} {
		t.Run(c.label, func(t *testing.T) {
			resp, err := parseFQDN(c.input)
			if err == nil && c.wantErr {
				t.Errorf("Did not receive an error, wanted an error: %v", err)
			}
			if err != nil && !c.wantErr {
				t.Errorf("Did not want an error, received na error: %v", err)
			}
			if err == nil {
				if resp.TLD != c.want.TLD || resp.SLD != c.want.SLD || resp.Subdomain != c.want.Subdomain {
					t.Errorf("Got %v want %v", resp, c.want)
				}
			}
		})
	}
}
