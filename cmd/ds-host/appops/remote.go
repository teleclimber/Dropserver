package appops

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"path"
	"sort"
	"strings"

	"code.dny.dev/ssrf"
	"github.com/blang/semver/v4"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

type RemoteAppGetter struct {
	Config        *domain.RuntimeConfig `checkinject:"required"`
	AppFilesModel interface {
		SavePackage(io.Reader) (string, error)
	} `checkinject:"required"`

	client *http.Client
}

func (r *RemoteAppGetter) init() {
	if r.client != nil {
		return
	}

	s := r.getSSRF()
	dialer := &net.Dialer{
		Control: s.Safe,
	}
	transport := &http.Transport{
		DialContext: dialer.DialContext,
	}
	r.client = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return errors.New("no redirects please")
		},
		Transport: transport,
	}
}

func (r *RemoteAppGetter) getSSRF() *ssrf.Guardian {
	prefixes4 := make([]netip.Prefix, 0)
	prefixes6 := make([]netip.Prefix, 0)
	for _, a := range r.Config.InternalNetwork.AllowedIPs {
		p := getPrefix(a)
		if p.Addr().Is4() {
			prefixes4 = append(prefixes4, p)
		} else if p.Addr().Is6() {
			prefixes6 = append(prefixes6, p)
		}
	}
	return ssrf.New(
		ssrf.WithPorts(443), // HTTPS only
		ssrf.WithAllowedV4Prefixes(prefixes4...),
		ssrf.WithAllowedV6Prefixes(prefixes6...))
}

func getPrefix(og string) netip.Prefix {
	a := og
	addr, err := netip.ParseAddr(a)
	if err == nil {
		if addr.Is4() {
			a = a + "/32"
		} else if addr.Is6() {
			a = a + "/128"
		}
	}
	p, err := netip.ParsePrefix(a)
	if err != nil {
		// this should never happen because these strings should be validated as parseable
		panic("unable to process allowed IP into prefix: " + og)
	}
	return p
}

// FetchListing fetches the listing and returns
// errors and warnings encountered
func (r *RemoteAppGetter) FetchListing(url string) (domain.AppListing, error) {
	r.init()

	resp, err := r.client.Get(url)
	if err != nil {
		// Error is of type url.Error: timeouts, etc...
		// who needs to know about this error?
		return domain.AppListing{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// Unclear what kind of error this would be ? Something like "Error reading response..."
		return domain.AppListing{}, err
	}

	// switch on resp status
	// - 404: tell user wrong URL
	// - basically 4xx and 5xx tell user with code
	// - 3xx redirects? How to handle? Does it depend on whether it's the initial fetch or re-fetch? (redirects are handled by the client defined above)
	// - 304 Not Modified will be relevant when we are re-checking for a changed listing
	// For now anything but 200 is an error (this will have to change)
	if resp.StatusCode != http.StatusOK {
		return domain.AppListing{}, fmt.Errorf("got response code: %v", resp.Status)
	}

	// check mime type? However it's not sure that a file server will serve JSON as application/json?

	var listing domain.AppListing
	err = json.Unmarshal(body, &listing)
	if err != nil {
		return domain.AppListing{}, err
	}

	return listing, nil
}

func ValidateListing(listing domain.AppListing) error {
	// TODO complete validation code

	// zero versions is a fail

	// Base URL should parse

	if len(listing.Versions) == 0 {
		return errors.New("there are no versions in this app listing")
	}

	//for i, v := range listing.Versions {
	// - validate each version as semver
	// - validate each relative path
	//   some can't be empty
	//   none can contain ..
	//}

	return nil
}

// URLFromListing returns a URL for a relative path of a listing
// It assumes values passed are already sanitzed.
func URLFromListing(listingURL, baseURL, relPath string) (string, error) {
	var u *url.URL
	var err error
	if baseURL == "" {
		u, err = url.Parse(listingURL)
		u.Path = path.Dir(u.Path)
	} else {
		u, err = url.Parse(baseURL)
	}
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, strings.TrimPrefix(relPath, "/"))
	return u.String(), nil
}

// FetchAppPackage from remote URL
// Creates location and returns locationKey as string
func (r *RemoteAppGetter) FetchPackageJob(url string) (string, error) {
	r.init()

	resp, err := r.client.Get(url)
	if err != nil {
		// Error is of type url.Error: timeouts, etc...
		// who needs to know about this error?
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("got response code: %s", resp.Status)
	}

	locationKey, err := r.AppFilesModel.SavePackage(resp.Body)
	return locationKey, err
}

type semverListing struct {
	semver *semver.Version
	verStr domain.Version
}

func GetLatestVersion(versions map[domain.Version]domain.AppListingVersion) (domain.Version, error) {
	if len(versions) == 0 {
		return domain.Version(""), errors.New("no versions in listing")
	}
	semVersions := make([]semverListing, len(versions))
	i := 0
	for v, _ := range versions {
		sver, err := semver.New(string(v))
		if err != nil {
			return domain.Version(""), err
		}
		semVersions[i] = semverListing{semver: sver, verStr: v}
		i++
	}

	sort.Slice(semVersions, func(i, j int) bool {
		return semVersions[i].semver.Compare(*semVersions[j].semver) == 1
	})

	return semVersions[0].verStr, nil

}
