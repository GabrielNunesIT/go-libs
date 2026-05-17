package cache

import (
	"context"
	"fmt"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/errors/errbase"
	"github.com/cockroachdb/errors/errorspb"
	"github.com/gogo/protobuf/proto"
)

type cacheError struct {
	operation string
	key       string
	message   string
}

var _ error = (*cacheError)(nil)
var _ fmt.Formatter = (*cacheError)(nil)
var _ errbase.SafeFormatter = (*cacheError)(nil)

func CacheErrorf(operation string, key string, message string) error {
	return &cacheError{operation: operation, key: key, message: message}
}

func (e *cacheError) Error() string {
	return fmt.Sprintf("cache error: %s failed for key %q: %s", e.operation, e.key, e.message)
}

func (e *cacheError) SafeDetails() []string {
	return []string{
		fmt.Sprintf("operation: %s", e.operation),
		fmt.Sprintf("key: %s", e.key),
		fmt.Sprintf("message: %s", e.message),
	}
}

func (e *cacheError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "cacheError{operation: %q, key: %q, message: %q}", e.operation, e.key, e.message)
			return
		}
		fallthrough
	case 's', 'q':
		fmt.Fprint(s, e.Error())
	}
}

func (e *cacheError) SafeFormatError(p errbase.Printer) (next error) {
	return nil
}

func init() {
	errors.RegisterLeafEncoder(errbase.GetTypeKey((*cacheError)(nil)), encodeCacheError)
	errors.RegisterLeafDecoder(errbase.GetTypeKey((*cacheError)(nil)), decodeCacheError)
}

func encodeCacheError(ctx context.Context, err error) (msg string, safeDetails []string, payload proto.Message) {
	var cacheErr *cacheError
	if errors.As(err, &cacheErr) {
		return cacheErr.Error(), cacheErr.SafeDetails(), &errorspb.StringPayload{Msg: cacheErr.message}
	}

	return err.Error(), nil, nil
}

func decodeCacheError(ctx context.Context, msg string, safeDetails []string, payload proto.Message) error {
	cacheErr := new(cacheError)
	for _, detail := range safeDetails {
		if after, ok := strings.CutPrefix(detail, "operation: "); ok {
			cacheErr.operation = after
		}
		if after, ok := strings.CutPrefix(detail, "key: "); ok {
			cacheErr.key = after
		}
		if after, ok := strings.CutPrefix(detail, "message: "); ok {
			cacheErr.message = after
		}
	}

	if payload != nil {
		if stringPayload, ok := payload.(*errorspb.StringPayload); ok {
			cacheErr.message = stringPayload.Msg
		}
	}

	return cacheErr
}
