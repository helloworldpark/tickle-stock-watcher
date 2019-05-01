package commons

import "fmt"

type errorWrapper struct {
	tag string
	msg string
}

// MetaError generates an error with a cached tag
type MetaError func(msg string) error

const errFormat = "[%s] %s"

func (err errorWrapper) Error() string {
	return fmt.Sprintf(errFormat, err.tag, err.msg)
}

// NewTaggedError generates an error generator given a tag
func NewTaggedError(tag string) MetaError {
	return func(msg string) error {
		return errorWrapper{tag: tag, msg: msg}
	}
}
