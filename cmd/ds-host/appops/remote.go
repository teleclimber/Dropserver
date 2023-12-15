package appops

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"time"

	"code.dny.dev/ssrf"
	"github.com/blang/semver/v4"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"github.com/teleclimber/DropServer/internal/nulltypes"
)

type RemoteAppGetter struct {
	Config        *domain.RuntimeConfig `checkinject:"required"`
	AppFilesModel interface {
		SavePackage(io.Reader) (string, error)
	} `checkinject:"required"`
	AppModel interface {
		GetAppUrlData(domain.AppID) (domain.AppURLData, error)
		GetAppUrlListing(domain.AppID) (domain.AppListing, domain.AppURLData, error)
		SetLastFetch(domain.AppID, time.Time, string) error
		SetListing(domain.AppID, domain.AppListingFetch) error
		SetNewUrl(domain.AppID, string, nulltypes.NullTime) error
		GetVersionsForApp(domain.AppID) ([]*domain.AppVersion, error)
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
			// hmm we may have different redirect rules for different tasks.
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

func (r *RemoteAppGetter) RefreshAppListing(appID domain.AppID) error {
	appUrlData, err := r.AppModel.GetAppUrlData(appID)
	if err != nil {
		return err
	}
	listingFetch, err := r.fetchListing(appUrlData.URL, appUrlData.Etag)
	if err != nil {
		err = fmt.Errorf("error fetching listing from %v: %w", appUrlData.URL, err)
		r.AppModel.SetLastFetch(appID, time.Now(), "error")
		return err
	}
	if listingFetch.NotModified {
		r.AppModel.SetLastFetch(appID, time.Now(), "not-modified")
		return nil
	}
	if getNewURL(listingFetch) != "" {
		r.AppModel.SetNewUrl(appID, getNewURL(listingFetch), nulltypes.NewTime(time.Now(), true))
		return errors.New("app listing has moved, new URL: " + getNewURL(listingFetch))
	}

	err = ValidateListing(listingFetch.Listing)
	if err != nil {
		r.AppModel.SetLastFetch(appID, time.Now(), "error")
		err = fmt.Errorf("error validating listing from %v: %w", appUrlData.URL, err)
		return err
	}

	latestVersion, err := GetLatestVersion(listingFetch.Listing.Versions)
	if err != nil {
		// this is a coding error: we validated the listing earlier so there is no reason to get an error here.
		r.AppModel.SetLastFetch(appID, time.Now(), "error")
		r.getLogger("RefreshAppListing").AddNote("Unexpected error fron GetLatestVersion").Error(err)
		return fmt.Errorf("unexpected error: %w", err)
	}
	listingFetch.LatestVersion = latestVersion

	r.AppModel.SetListing(appID, listingFetch)

	return nil
}

func (r *RemoteAppGetter) FetchValidListing(url string) (domain.AppListingFetch, error) {
	listingFetch, err := r.fetchListing(url, "")
	if err != nil {
		err = fmt.Errorf("error fetching listing from %v: %w", url, err)
		return domain.AppListingFetch{}, err
	}
	if listingFetch.NotModified {
		// this shouldn't happen since we didn't send an etag, but we can't carry on.
		return domain.AppListingFetch{}, errors.New("error fetching listing: endpoint returned last-modified, but we didn't send an etag")
	}
	if getNewURL(listingFetch) != "" {
		return listingFetch, nil
	}

	err = ValidateListing(listingFetch.Listing)
	if err != nil {
		err = fmt.Errorf("error validating listing from %v: %w", url, err)
		return domain.AppListingFetch{}, err
	}

	latestVersion, err := GetLatestVersion(listingFetch.Listing.Versions)
	if err != nil {
		// this is a coding error: we validated the listing earlier so there is no reason to get an error here.
		r.getLogger("FetchValidListing").AddNote("Unexpected error fron GetLatestVersion").Error(err)
		return domain.AppListingFetch{}, fmt.Errorf("unexpected error: %w", err)
	}
	listingFetch.LatestVersion = latestVersion

	return listingFetch, nil
}

// fetchListing fetches the listing and returns
// errors and warnings encountered
func (r *RemoteAppGetter) fetchListing(url string, etag string) (domain.AppListingFetch, error) {
	r.init()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return domain.AppListingFetch{}, err
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		// Error is of type url.Error: timeouts, etc...
		// who needs to know about this error?
		return domain.AppListingFetch{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(&io.LimitedReader{R: resp.Body, N: domain.AppListingMaxFileSize})
	if err != nil {
		// Unclear what kind of error this would be ? Something like "Error reading response..."
		return domain.AppListingFetch{}, err
	}

	// switch on resp status
	// - 404: tell user wrong URL
	// - basically 4xx and 5xx tell user with code
	// - 3xx redirects? How to handle? Does it depend on whether it's the initial fetch or re-fetch? (redirects are handled by the client defined above)
	// - 304 Not Modified will be relevant when we are re-checking for a changed listing
	// For now anything but 200 is an error (this will have to change)
	if resp.StatusCode == http.StatusNotModified {
		return domain.AppListingFetch{NotModified: true}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return domain.AppListingFetch{}, fmt.Errorf("got response code: %v", resp.Status)
	}

	newEtag := resp.Header.Get("ETag")
	lm := resp.Header.Get("Last-Modified")
	newLastModified := time.Now()
	if lm != "" {
		lmt, err := http.ParseTime(lm)
		if err != nil {
			// This is kind of a pain to deal with. Not problematic enough to kill the whole process
			// But user should know of issue so that info can be relayed to developer?
			// no-op for now.
		} else {
			newLastModified = lmt
		}
	}

	// check mime type? However it's not sure that a file server will serve JSON as application/json?

	var listing domain.AppListing
	err = json.Unmarshal(body, &listing)
	if err != nil {
		return domain.AppListingFetch{}, err
	}

	return domain.AppListingFetch{
		Listing:         listing,
		Etag:            newEtag,
		ListingDatetime: newLastModified}, nil
}

func (r *RemoteAppGetter) FetchNewVersionManifest(appID domain.AppID, version domain.Version) (domain.AppGetMeta, error) {
	listing, urlData, err := r.AppModel.GetAppUrlListing(appID)
	if err != nil {
		return domain.AppGetMeta{}, err
	}

	// check url data for reasons to bail: like new url?

	// bail if no versions:
	if len(listing.Versions) == 0 {
		return domain.AppGetMeta{
			AppID: appID,
			Warnings: []domain.ProcessWarning{{
				Field:   "listing",
				Problem: domain.ProblemInvalid,
				Message: "App listing contains zero versions",
				// TODO mark as Fatal
			}},
		}, nil
	}

	// if version is not set, return the latest version:
	if version == domain.Version("") {
		version, err = GetLatestVersion(listing.Versions)
		if err != nil {
			return domain.AppGetMeta{
				AppID: appID,
				Warnings: []domain.ProcessWarning{{
					Field:   "listing",
					Problem: domain.ProblemError,
					Message: "Error while determining latest version in app listing: " + err.Error(),
					// TODO mark as Fatal
				}},
			}, nil
		}
	}

	manifest, err := r.fetchManifestFromListing(urlData.URL, listing, version)
	if err != nil {
		if err != nil {
			return domain.AppGetMeta{
				AppID: appID,
				Warnings: []domain.ProcessWarning{{
					Field:   "manifest",
					Problem: domain.ProblemError,
					Message: "Error while fetching the app manifest: " + err.Error(),
					// TODO mark as Fatal
				}},
			}, nil
		}
	}

	manifest, warnings := validateManifest(manifest)

	appVersions, err := r.getVersionSemvers(appID)
	if err != nil {
		return domain.AppGetMeta{}, err
	}
	prev, next, warns := validateVersionSequence(manifest.Version, manifest.Schema, appVersions)
	warnings = addWarning(warnings, warns...)

	return domain.AppGetMeta{
		AppID:           appID,
		PrevVersion:     prev,
		NextVersion:     next,
		Warnings:        warnings,
		VersionManifest: manifest,
		// TODO: convey errors as well, but requires more refactoring
	}, nil
}

func (r *RemoteAppGetter) getVersionSemvers(appID domain.AppID) ([]appVersionSemver, error) {
	appVersions, err := r.AppModel.GetVersionsForApp(appID)
	if err != nil {
		return nil, err
	}
	ret := make([]appVersionSemver, len(appVersions))
	for i, appVersion := range appVersions {
		sver, err := semver.Parse(string(appVersion.Version))
		if err != nil {
			r.getLogger("getVersionSemvers() semver.Parse").AppID(appID).AddNote(fmt.Sprintf("bad version: %v", appVersion.Version)).Error(err)
			return nil, err
		}
		ret[i] = appVersionSemver{*appVersion, sver}
	}
	return ret, nil
}

// FetchManifest fetches the manifest given ... ?
// This should be used when we don't have an app installed.
// So what's the reference?
// We need the original listing URL, the listing Base, and the relative path of the version.
// But sending the listing back up from the frontend is kind of lame?
// If the listing was cached, we could just take the url + the version, re-request the listing and that gives everything we need.
// Not bad, now?
// Plus it would work for the initial request
func (r *RemoteAppGetter) FetchManifest(listingURL string, version domain.Version) (domain.AppVersionManifest, []domain.ProcessWarning, error) {
	// Here it would be good to have an HTP cache of some sort on listing,
	listingFetch, err := r.fetchListing(listingURL, "")
	if err != nil {
		return domain.AppVersionManifest{}, []domain.ProcessWarning{}, err
	}

	manifest, err := r.fetchManifestFromListing(listingURL, listingFetch.Listing, version)
	if err != nil {
		return domain.AppVersionManifest{}, []domain.ProcessWarning{}, err
	}

	manifest, warnings := validateManifest(manifest)

	return manifest, warnings, nil
}

func (r *RemoteAppGetter) fetchManifestFromListing(listingURL string, listing domain.AppListing, version domain.Version) (domain.AppVersionManifest, error) {
	versionListing, ok := listing.Versions[version]
	if !ok {
		return domain.AppVersionManifest{}, errors.New("no such version in app listing")
	}

	manifestURL, err := URLFromListing(listingURL, listing.Base, versionListing.Manifest)
	if err != nil {
		r.getLogger("fetchManifestFromListing(), URLFromListing()").Error(err)
		return domain.AppVersionManifest{}, err
	}

	manifest, err := r.fetchManifest(manifestURL)

	return manifest, err
}

func (r *RemoteAppGetter) fetchManifest(url string) (domain.AppVersionManifest, error) {
	var manifest domain.AppVersionManifest
	r.init()
	resp, err := r.client.Get(url)
	if err != nil {
		// Error is of type url.Error: timeouts, etc...
		// who needs to know about this error?
		return manifest, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return manifest, fmt.Errorf("got response code: %s", resp.Status)
	}
	body, err := io.ReadAll(&io.LimitedReader{R: resp.Body, N: domain.AppManifestMaxFileSize})
	if err != nil {
		// Unclear what kind of error this would be ? Something like "Error reading response..."
		return manifest, err
	}

	err = json.Unmarshal(body, &manifest)

	return manifest, err
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

func (g *RemoteAppGetter) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("RemoteAppGetter")
	if note != "" {
		l.AddNote(note)
	}
	return l
}

func getNewURL(listingFetch domain.AppListingFetch) string {
	if listingFetch.NewURL != "" {
		return listingFetch.NewURL
	} else if listingFetch.Listing.NewURL != "" {
		return listingFetch.Listing.NewURL
	}
	return ""
}
