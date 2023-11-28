package appops

import (
	"fmt"
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestURLFromListing(t *testing.T) {
	cases := []struct {
		listingURL string
		base       string
		rel        string
		out        string
	}{{
		listingURL: "site.com/listing.json",
		rel:        "package.tar.gz",
		out:        "site.com/package.tar.gz",
	}, {
		listingURL: "site.com/deep/path/listing.json",
		rel:        "package.tar.gz",
		out:        "site.com/deep/path/package.tar.gz",
	}, {
		listingURL: "site.com/",
		rel:        "package.tar.gz",
		out:        "site.com/package.tar.gz",
	}, {
		listingURL: "site.com",
		base:       "site.com",
		rel:        "package.tar.gz",
		out:        "site.com/package.tar.gz",
	}, {
		listingURL: "https://site.com/listing.json",
		rel:        "package.tar.gz",
		out:        "https://site.com/package.tar.gz",
	}, {
		listingURL: "https://site.com/listing.json",
		rel:        "package.tar.gz\n",
		out:        "https://site.com/package.tar.gz%0A", // the bad char in path should get url-encoded.
	}}

	for i, c := range cases {
		t.Run(fmt.Sprintf("case %v", i), func(t *testing.T) {
			result, err := URLFromListing(c.listingURL, c.base, c.rel)
			if err != nil {
				t.Error(err)
			}
			if result != c.out {
				t.Errorf("got %v expected %v", result, c.out)
			}
		})
	}
}

func TestGetLatestVersion(t *testing.T) {
	versions := map[domain.Version]domain.AppListingVersion{}
	versions["0.1.0"] = domain.AppListingVersion{}
	versions["1.2.3"] = domain.AppListingVersion{}
	versions["0.1.1"] = domain.AppListingVersion{}

	v, err := GetLatestVersion(versions)
	if err != nil {
		t.Fatal(err)
	}
	if v != domain.Version("1.2.3") {
		t.Errorf("expected 1.2.3, got %v", v)
	}
}

func TestValidateListingPath(t *testing.T) {
	cases := []struct {
		desc     string
		path     string
		required bool
		fail     bool
	}{{
		desc:     "empty path, not required",
		path:     "",
		required: false,
		fail:     false,
	}, {
		desc:     "empty path, required",
		path:     "",
		required: true,
		fail:     true,
	}, {
		desc: "path",
		path: "/abc/def",
		fail: false,
	}}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			err := validateListingPath(c.path, c.required)
			if c.fail && err == nil {
				t.Error("expected vaildation to fail")
			} else if !c.fail && err != nil {
				t.Errorf("expectted validation to pass, got error: %v", err)
			}
		})
	}
}

func TestValidateListing(t *testing.T) {
	cases := []struct {
		desc    string
		listing domain.AppListing
		fail    bool
	}{{
		desc:    "empty listing",
		listing: domain.AppListing{},
		fail:    true,
	}, {
		desc:    "empty listing with new url",
		listing: domain.AppListing{NewURL: "abc.com/app"},
		fail:    false,
	}, {
		desc: "one version",
		listing: domain.AppListing{
			Versions: map[domain.Version]domain.AppListingVersion{
				domain.Version("1.2.3"): {Package: "pack.tar.gz", Manifest: "manifest.json"},
			},
		},
		fail: false,
	}, {
		desc: "one version bad base",
		listing: domain.AppListing{
			Base: "\n", // ascii control character makes URL invalid
			Versions: map[domain.Version]domain.AppListingVersion{
				domain.Version("1.2.3"): {Package: "pack.tar.gz", Manifest: "manifest.json"},
			},
		},
		fail: true,
	}, {
		desc: "one version missing package path",
		listing: domain.AppListing{
			Versions: map[domain.Version]domain.AppListingVersion{
				domain.Version("1.2.3"): {Manifest: "manifest.json"},
			},
		},
		fail: true,
	}}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			err := ValidateListing(c.listing)
			if c.fail && err == nil {
				t.Error("expected vaildation to fail")
			} else if !c.fail && err != nil {
				t.Errorf("expectted validtaion to pass, got error: %v", err)
			}
		})
	}
}
