package main

import (
	"fmt"
	nc "github.com/billputer/go-namecheap"
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

func TestUpdateDomain(t *testing.T) {
	d := newClient("test", "testing-token", "Testing McTester")
	for _, c := range []struct {
		fqdn      string
		ipAddress string
		wantErr   bool
	}{
		{
			fqdn:      "somehost.sld.tld",
			ipAddress: "127.0.0.1",
			wantErr:   true,
		},
		{
			fqdn:      "somehost.sld.tld",
			ipAddress: "123.13.31.30",
			wantErr:   false,
		},
	} {
		err := updateDomain(d, c.fqdn, c.ipAddress)
		if err != nil && !c.wantErr {
			t.Errorf("Got error, didn't want error: %v", err)
		}
		if err == nil && c.wantErr {
			t.Errorf("Did not get an error, wanted an error.")
		}
	}
}

func TestReverse(t *testing.T) {
	for _, c := range []struct {
		input []string
		want  []string
	}{
		{
			input: []string{"one", "two", "three"},
			want:  []string{"three", "two", "one"},
		},
		{
			input: []string{"abc", "def", "ghi", "jkl"},
			want:  []string{"jkl", "ghi", "def", "abc"},
		},
	} {
		res := reverse(c.input)
		for k, v := range res {
			if c.want[k] != v {
				t.Errorf("Got %v, want %v", res, c.want)
			}
		}
	}
}

func TestParseFQDN(t *testing.T) {
	for _, c := range []struct {
		input   string
		want    fqdn
		wantErr bool
	}{
		{
			input: "somehost.corp.com",
			want: fqdn{
				TLD:       "com",
				SLD:       "corp",
				Subdomain: "somehost",
			},
			wantErr: false,
		},
		{
			input:   "google.com",
			wantErr: true,
		},
		{
			input: "sub.subdomain.foo.bar",
			want: fqdn{
				TLD:       "bar",
				SLD:       "foo",
				Subdomain: "subdomain",
			},
			wantErr: false,
		},
	} {
		resp, err := parseFQDN(c.input)
		if err != nil && c.wantErr {
			continue
		}
		if err == nil && c.wantErr {
			t.Errorf("Did not receive an error, wanted an error: %v", err)
			continue
		}
		if resp.TLD != c.want.TLD || resp.SLD != c.want.SLD || resp.Subdomain != c.want.Subdomain {
			t.Errorf("Got %v want %v", resp, c.want)
		}
	}
}
