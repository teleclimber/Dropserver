package appops

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// functions to fetch things from an app distribution site
// as well parse/interpret / validate the incoming data unless it's done elsewhere.

type RemoteAppGetter struct {
	AppFilesModel interface {
		SavePackage(io.Reader) (string, error)
	} `checkinject:"required"`
}

// FetchListing fetches the listing and returns
// errors and warnings encountered
func (r *RemoteAppGetter) FetchListing(url string) (domain.AppListing, error) { // return a warning or not???
	// What should the redirect policy be?
	// - if just http > https then fine.
	// - otherwise question it?

	// TODO: SSRF protection and don't use default client and don't create a new client each time!
	// Also, passing the client makes it possible to test this function?

	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		// Error is of type url.Error: timeouts, etc...
		// whow needs to know about this error?
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
	resp, err := http.DefaultClient.Get(url) // TODO SSRF protection
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
