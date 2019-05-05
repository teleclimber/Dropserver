package dserror

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

var stdRegex = regexp.MustCompile(`^ds-error:([0-9]+):(.*)`)

// Error reports an error that the end-user can do somthing about
type Error struct {
	code         domain.ErrorCode
	extraMessage string
}

// PublicString returns the error message that can be seen by users
func (e Error) PublicString() string {
	errString := ""
	msg, ok := errorMesage[e.code]
	if ok {
		errString = msg
	} else {
		errString = fmt.Sprintf("User Error %d", e.code)
	}

	if e.code != InternalError {
		errString = errString + " " + e.extraMessage
	}

	return errString
}

// Code returns the error code
func (e Error) Code() domain.ErrorCode {
	return e.code
}

// ExtraMessage returns the extra message
func (e Error) ExtraMessage() string {
	return e.extraMessage
}

// HTTPError sends the error to the response writer
func (e Error) HTTPError(res http.ResponseWriter) {
	http.Error(res, e.PublicString(), e.getHTTPStatus())
}

func (e Error) getHTTPStatus() int {
	code, ok := httpCode[e.code]
	if ok {
		return code
	}
	return http.StatusBadRequest
}

// ToStandard returns an error of type error with the ds-error code in the string
// You can then use FromStandard to get the original error back
// Handy for when an interface requires the use of error (like net/rpc)
func (e Error) ToStandard() error {
	return fmt.Errorf("ds-error:%d:%s", e.code, e.extraMessage)
}

// New returns a user error with specified code
func New(code domain.ErrorCode, extraMessages ...string) Error {
	var extra string
	if len(extraMessages) > 0 {
		extra = extraMessages[0]
	}
	return Error{
		code:         code,
		extraMessage: extra }
}

// FromStandard takes a regular error type and turns it into ds error
func FromStandard(err error) Error {
	matches := stdRegex.FindStringSubmatch(err.Error())
	if len(matches) > 0 {
		codeInt, err := strconv.Atoi(matches[1])
		if err != nil {
			return New(InternalError, err.Error())
		}
		code := domain.ErrorCode(codeInt)
		return New(code, matches[2])
	}
	return New(InternalError, err.Error())
}

// Encoded tells you if an error is an encoded ds-error
// Useful if you need to know whether to handle a stanard error
// or if it was likely handled when it was made into a dserror
func Encoded(err error) bool {
	matches := stdRegex.FindStringSubmatch(err.Error())
	return len(matches) > 0
}

// we might have a method that writes the 4xx response?

// should we catalog all errors and attach them to an int, kind of like log levels?
// See https://golang.org/src/net/http/status.go for one way to do this

// what code numerical codes look like?
// -> integers, with digits divided in pairs starting to the right: 101 -> 1 ~ 01
// -> left most pair is general area, while rightmost pair is the message number within area.
// If you run out of digits within a pair, extend the int by two digits ("grow a pair")

// 10xx not found, plain bad requests, basics
// 20xx auth errors
// 30xx user route errors
// 32xx application route errors
// 34xx appspace management route errors
// 50xx admin route errors
// 70xx app-space route errors

const (
	// InternalError is a special error code that will not print its extra messages
	InternalError        domain.ErrorCode = 1
	AppConfigNotFound    domain.ErrorCode = 3201
	AppConfigParseFailed domain.ErrorCode = 3202
	AppConfigProblem 	 domain.ErrorCode = 3203
)

var errorMesage = map[domain.ErrorCode]string{
	InternalError:        "Internal Error",
	AppConfigNotFound:    "Could not find application.json",
	AppConfigParseFailed: "Failed to parse application.json",
	AppConfigProblem:     "Problem in application.json",
}

var httpCode = map[domain.ErrorCode]int{
	InternalError: http.StatusInternalServerError,
}

