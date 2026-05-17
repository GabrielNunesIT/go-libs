package startup

import (
	"context"
	"fmt"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/errors/errbase"
	"github.com/cockroachdb/errors/errorspb"
	"github.com/gogo/protobuf/proto"
)

type initializationError struct {
	component string
	message   string
}

var _ error = (*initializationError)(nil)
var _ fmt.Formatter = (*initializationError)(nil)
var _ errbase.SafeFormatter = (*initializationError)(nil)

func InitializationErrorf(component string, message string) error {
	return &initializationError{component: component, message: message}
}

func (e *initializationError) Error() string {
	return fmt.Sprintf("initialization error: %s failed to initialize: %s", e.component, e.message)
}

func (e *initializationError) SafeDetails() []string {
	return []string{
		fmt.Sprintf("component: %s", e.component),
		fmt.Sprintf("message: %s", e.message),
	}
}

func (e *initializationError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "initializationError{component: %q, message: %q}", e.component, e.message)
			return
		}
		fallthrough
	case 's', 'q':
		fmt.Fprint(s, e.Error())
	}
}

func (e *initializationError) SafeFormatError(p errbase.Printer) (next error) {
	return nil
}

func init() {
	errors.RegisterLeafEncoder(errbase.GetTypeKey((*initializationError)(nil)), encodeInitializationError)
	errors.RegisterLeafDecoder(errbase.GetTypeKey((*initializationError)(nil)), decodeInitializationError)
}

func encodeInitializationError(ctx context.Context, err error) (msg string, safeDetails []string, payload proto.Message) {
	var initErr *initializationError
	if errors.As(err, &initErr) {
		return initErr.Error(), initErr.SafeDetails(), &errorspb.StringPayload{Msg: initErr.message}
	}

	return err.Error(), nil, nil
}

func decodeInitializationError(ctx context.Context, msg string, safeDetails []string, payload proto.Message) error {
	initErr := new(initializationError)
	for _, detail := range safeDetails {
		if after, ok := strings.CutPrefix(detail, "component: "); ok {
			initErr.component = after
		}
		if after, ok := strings.CutPrefix(detail, "message: "); ok {
			initErr.message = after
		}
	}

	if payload != nil {
		if stringPayload, ok := payload.(*errorspb.StringPayload); ok {
			initErr.message = stringPayload.Msg
		}
	}

	return initErr
}
