package main

import (
	"testing"
)

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
		        TLD: "bar",
		        SLD: "foo",
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
