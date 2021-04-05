package ds2ds

import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// dropserver to dropserver comms.
// encapsulates:
// - mTLS verification (if enabled)
// - send a new request as a response

// Comms:
// - remote appspace login
// - user added to appspace (or change)
// - versioning of API (maybe through version hashes in header?)
// - ...

// Regarding the request/response paradigm
// First, not convinced we should implement it on this first pass.
// Second, it's not so straightforward:
// - If requesting a token, it's the token that is salient so it should be sent as a request.
// - If claiming a user is added to an appspace, what is salient?
//   .. maybe a verification has to be sent directly to appspace?
// --> worry about having requests result in more requests, because that is an avenue for DDOS
//   --> yes, but mTLS will largely defeat that

// This might be a good time to introduce go Context to our code , which is so far incredibly devoid of them.

// What are the incoming requests?
// [to: appspace domain] please send a login token for user
// [drop id domain] here is a login token?
// ...

// What are all the different request paths, and where are they coming from, and where are they going
// (in addition to ds2ds paths)
// - avatars: fetch from app frontend to a different domain so access can be limited (although is there any point to that?)
// -

// Note that inside appspacreouterpackage we have a dropserverroutes http handler
// ..that is not implemented but is said to serve login token requests.

type DS2DS struct {
	Config *domain.RuntimeConfig
}

func (d *DS2DS) GetRemoteAPIVersion(domainName string) (int, error) {
	// check if we stored the last used API version for this address in DB
	// If so return that
	// If not hit the remote's /.dropserver/api-version-check or whatever end point and find a match
	// (and stash the result)

	// for now fake it til you make it
	return 0, nil
}
