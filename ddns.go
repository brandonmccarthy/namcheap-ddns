package main

import (
	"flag"
	"fmt"
	namecheap "github.com/billputer/go-namecheap"
	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var IP_REGEX = regexp.MustCompile(`[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}`)

type fqdn struct {
	TLD, SLD, Subdomain string
}

type apiAuth struct {
	User, Username, Token string
}

// updateDomain uses the namecheap API to set the record of a host
func updateDomain(a apiAuth, f, ip string) error {
	domain, err := parseFQDN(f)
	if err != nil {
		return fmt.Errorf("unable to parse FQDN: %v", err)
	}

	var req []namecheap.DomainDNSHost
	req = append(req, namecheap.DomainDNSHost{
		Name:    domain.Subdomain,
		Type:    "A",
		Address: ip,
	},
	)

	client := namecheap.NewClient(a.User, a.Token, a.Username)
	res, err := client.DomainDNSSetHosts(domain.SLD, domain.TLD, req)
	if err != nil {
		return fmt.Errorf("unable to set host address: %v", err)
	}
	if !res.IsSuccess {
		return fmt.Errorf("dns change request unsuccessful")
	}
	return nil
}

// Lookup the ip addresses of a domain and return list of IP addresses
func getIPAddresses(domain string) ([]net.IP, error) {
	resp, err := net.LookupIP(domain)
	if err != nil {
		return nil, fmt.Errorf("unable to lookup domain %s: %v\n", domain, err)
	}
	return resp, nil
}

// Reverse slice of string
func reverse(s []string) []string {
	var result []string
	for i := len(s) - 1; i >= 0; i-- {
		result = append(result, s[i])
	}
	return result
}

// parseFQDN takes a string and returns fqdn struct
func parseFQDN(f string) (*fqdn, error) {
	fs := strings.Split(f, ".")
	fs = reverse(fs)
	if len(fs) < 3 {
		return nil, fmt.Errorf("FQDN must have TLD, SLD, and subdomain")
	}
	return &fqdn{
		TLD:       fs[0],
		SLD:       fs[1],
		Subdomain: fs[2],
	}, nil
}

// getLocalIP uses an external website to determine local public IP address and
// return the net.IP type address
func getLocalIP(ipResolver string) (net.IP, error) {
	var resp *http.Response
	operation := func() error {
		r, err := http.Get(ipResolver)
		if err != nil {
			return &backoff.PermanentError{Err: fmt.Errorf("error accessing URL %s: %v\n", ipResolver, err)}
		}
		if r.StatusCode != 200 {
			glog.Infof("Domain %s responded with status code %d, retrying...", ipResolver, r.StatusCode)
			return fmt.Errorf("server returned error code %d\n", resp.StatusCode)
		}
		resp = r
		return nil
	}

	err := backoff.Retry(operation, backoff.NewExponentialBackOff())
	if err != nil {
		return nil, fmt.Errorf("unable to get response from server: %v\n", err)
	}

	defer resp.Body.Close()
	bodyResp, err := ioutil.ReadAll(resp.Body)
	body := string(bodyResp)
	if err != nil {
		return nil, fmt.Errorf("error reading body response: %v\n", err)
	}

	match := IP_REGEX.FindString(body)
	if match == "" {
		return nil, fmt.Errorf("regex couldn't find IP address in website response\n")
	}

	ipAddr := net.ParseIP(match)
	if ipAddr == nil {
		return nil, fmt.Errorf("unable to parse IP address from regex match\n")
	}
	return ipAddr, nil
}

func main() {
	// What is the difference between user and username?
	var (
		domainList  = flag.String("domains", "", "The domains we want to check against.")
		ipResolver  = flag.String("resolver", "http://canhazip.com", "The full URL for the domain to resolve local IP against.")
		apiUser     = flag.String("user", "", "Username to use for the namecheap API.")
		apiToken    = flag.String("token", "", "Token for the namecheap API.")
		apiUsername = flag.String("username", "", "Name to use for the namecheap API.")
		tts         = flag.Duration("sleep", 30*time.Minute, "Duration to sleep between checking localIP and domains.")
	)

	flag.Parse()
	var domains []string

	// Variable checking to make sure we have values
	if *domainList == "" {
		glog.Exitf("No domains given, exiting.")
	} else {
		domains = strings.Split(*domainList, ",")
	}
	if *apiUser == "" {
		glog.Exitf("No user for namecheap API specified, exiting.")
	}
	if *apiToken == "" {
		glog.Exitf("No token for namecheap API specified, exiting.")
	}
	if *apiUsername == "" {
		glog.Exitf("No username for namecheap API specified, exiting.")
	}

	nAPI := apiAuth{
		User:     *apiUser,
		Username: *apiUsername,
		Token:    *apiToken,
	}

	// Run until interrupted
	for {
		// Get the local IP address and error if not
		glog.Info("Getting local IP address")
		localIP, err := getLocalIP(*ipResolver)
		if err != nil {
			glog.Errorf("Unable to get local IP address: %v\n", err)
		}
		glog.Infof("Local IP address is %s\n", localIP.String())

		// Iterate through domains and check IP addresses and update if needed
		for _, domain := range domains {
			domainIPs, err := getIPAddresses(domain)
			if err != nil {
				glog.Warningf("Could not get IP addresses for domain %s: %v\n", domain, err)
			}
			domainIP := domainIPs[0] // I only have one IP address per sub-domain
			if domainIP.String() != localIP.String() {
				glog.Warningf("Domain %s has IP %s want %s\n", domain, domainIP.String(), localIP.String())
				err := updateDomain(nAPI, domain, localIP.String())
				if err != nil {
					glog.Warningf("Could not update domain %s: %v", domain, err)
				}
			} else {
				glog.Infof("Domain %s is good", domain)
			}
		}

		glog.Infof("Starting to sleep for %v", *tts)
		time.Sleep(*tts)
	}
}