package validator

import "strings"

// Normalizations are there to ensure that strings representing
// unique thingscan be compared directly.

// NormalizeEmail makes emails comparable
func NormalizeEmail(email string) string {
	return strings.ToLower(email)
}

// NormalizeDomainName makes domain names comparable
func NormalizeDomainName(domainName string) string {
	// just lowercpas for now
	// might look at eliminating protocol, etc...
	// trailing / leading dots?

	return strings.ToLower(domainName)
}

// NormalizeDropIDHandle makes handles comparable
func NormalizeDropIDHandle(h string) string {
	return strings.ToLower(h)
}

//NormalizeDropIDFull makes full dropid strings comparable
func NormalizeDropIDFull(dropid string) string {
	h, d := SplitDropID(dropid)

	h = NormalizeDropIDHandle(h)
	d = NormalizeDomainName(d)

	return JoinDropID(h, d)
}

// NormalizeDisplayName makes display names better
func NormalizeDisplayName(dn string) string {
	return strings.TrimSpace(dn)
}
