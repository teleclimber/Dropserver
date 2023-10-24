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
