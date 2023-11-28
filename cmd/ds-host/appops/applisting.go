package appops

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func ValidateListing(listing domain.AppListing) error { // Move to a new file or turn into a method of RemoteAppGetter.
	if listing.NewURL != "" {
		// If NewURL is set, we ignore the rest of the listing
		// We don't validate the new url at this stage.
		return nil
	}

	if listing.Base != "" {
		_, err := url.Parse(listing.Base)
		if err != nil {
			return fmt.Errorf("base url did not parse: %w", err)
		}
	}

	if len(listing.Versions) == 0 {
		return errors.New("there are no versions in this app listing")
	}

	for v, vd := range listing.Versions {
		_, err := semver.Parse(string(v))
		if err != nil {
			return fmt.Errorf("failed to parse version %v: %w", v, err)
		}
		err = validateListingPath(vd.Manifest, true)
		if err != nil {
			return fmt.Errorf("failed to validate manifest path for version %v: %w", v, err)
		}
		err = validateListingPath(vd.Package, true)
		if err != nil {
			return fmt.Errorf("failed to validate package path for version %v: %w", v, err)
		}
		err = validateListingPath(vd.Changelog, false)
		if err != nil {
			return fmt.Errorf("failed to validate changelog path for version %v: %w", v, err)
		}
		err = validateListingPath(vd.Icon, false)
		if err != nil {
			return fmt.Errorf("failed to validate icon path for version %v: %w", v, err)
		}
	}

	return nil
}
func validateListingPath(p string, required bool) error {
	p = strings.TrimSpace(p)
	if p == "" {
		if !required {
			return nil
		}
		return fmt.Errorf("required path is empty")
	}

	return nil
}

// URLFromListing returns a URL for a relative path of a listing
// It assumes values passed are already sanitzed.
func URLFromListing(listingURL, baseURL, relPath string) (string, error) { // leave here ot move to independent remote function?
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
	for v := range versions {
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
