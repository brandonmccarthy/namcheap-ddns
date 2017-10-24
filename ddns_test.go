package main

import (
	"context"
	"fmt"
	nc "github.com/billputer/go-namecheap"
	"net/http"
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
func mockHTTPServer(label, input string, t *testing.T) {
	t.Log("Creating server to listen on 127.0.0.1:8080")
	stop := make(chan bool)
	srv := &http.Server{
		Addr: ":8080",
	}
	t.Log("Creating handler function for HTTP server")
	http.HandleFunc("/"+label, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, input)
		stop <- true
	})
	go func(srv *http.Server) {
		t.Logf("%v\n", srv.ListenAndServe())
	}(srv)
	<-stop
	t.Log("Received request from server to shutdown")
	ctx := context.Background()
	srv.Shutdown(ctx)
}

func TestGetLocalIP(t *testing.T) {
	for _, c := range []struct {
		label   string
		url     string
		input   string
		want    string
		wantErr bool
	}{
		{
			label:   "success",
			url:     "http://localhost:8080/success",
			input:   "192.168.10.1\n\n",
			want:    "192.168.10.1",
			wantErr: false,
		},
		{
			label:   "hardparse",
			url:     "http://localhost:8080/hardparse",
			input:   "Your IP address is: 192.168.30.1\n\n",
			want:    "192.168.30.1",
			wantErr: false,
		},
		{
			label:   "failure",
			url:     "http://lcaolhost:8080/failure",
			input:   "EROROROORORO\n\n",
			want:    "nil",
			wantErr: true,
		},
	} {
		go mockHTTPServer(c.label, c.input, t)
		ip, err := getLocalIP(c.url)
		if err != nil && c.wantErr {
			continue
		}
		if err != nil && !c.wantErr {
			t.Errorf("Test %s failed: received error, did not want error: %v", c.label, err)
			continue
		}
		if err == nil && c.wantErr {
			t.Errorf("Test %s failed: did not receive error, wanted an error", c.label)
			continue
		}
		if ip.String() != c.want {
			t.Errorf("Test %s failed: got %s, wanted %s", c.label, ip.String(), c.want)
		} else {
			t.Logf("Test %s passed: got %s, wanted %s", c.label, ip.String(), c.want)
		}
		
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
		err := updateDomain(d, c.fqdn, c.ipAddress)
		if err != nil && !c.wantErr {
			t.Errorf("Test %s failed: got error, didn't want error: %v", c.label, err)
		}
		if err == nil && c.wantErr {
			t.Errorf("Test %s failed: did not get an error, wanted an error.", c.label)
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
				t.Errorf("Test failed: Got %v, want %v", res, c.want)
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
