package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/errors/errbase"
	"github.com/cockroachdb/errors/errorspb"
	"github.com/gogo/protobuf/proto"
)

type configurationError struct {
	field   string
	issue   string
	message string
}

var _ error = (*configurationError)(nil)
var _ fmt.Formatter = (*configurationError)(nil)
var _ errbase.SafeFormatter = (*configurationError)(nil)

func ConfigurationErrorf(field string, issue string, message string) error {
	return &configurationError{field: field, issue: issue, message: message}
}

func (e *configurationError) Error() string {
	return fmt.Sprintf("configuration error: field %q is %s: %s", e.field, e.issue, e.message)
}

func (e *configurationError) SafeDetails() []string {
	return []string{
		fmt.Sprintf("field: %s", e.field),
		fmt.Sprintf("issue: %s", e.issue),
		fmt.Sprintf("message: %s", e.message),
	}
}

func (e *configurationError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "configurationError{field: %q, issue: %q, message: %q}", e.field, e.issue, e.message)
			return
		}
		fallthrough
	case 's', 'q':
		fmt.Fprint(s, e.Error())
	}
}

func (e *configurationError) SafeFormatError(p errbase.Printer) (next error) {
	return nil
}

func init() {
	errors.RegisterLeafEncoder(errbase.GetTypeKey((*configurationError)(nil)), encodeConfigurationError)
	errors.RegisterLeafDecoder(errbase.GetTypeKey((*configurationError)(nil)), decodeConfigurationError)
}

func encodeConfigurationError(ctx context.Context, err error) (msg string, safeDetails []string, payload proto.Message) {
	var configErr *configurationError
	if errors.As(err, &configErr) {
		return configErr.Error(), configErr.SafeDetails(), &errorspb.StringPayload{Msg: configErr.message}
	}

	return err.Error(), nil, nil
}

func decodeConfigurationError(ctx context.Context, msg string, safeDetails []string, payload proto.Message) error {
	configErr := new(configurationError)
	for _, detail := range safeDetails {
		if after, ok := strings.CutPrefix(detail, "field: "); ok {
			configErr.field = after
		}
		if after, ok := strings.CutPrefix(detail, "issue: "); ok {
			configErr.issue = after
		}
		if after, ok := strings.CutPrefix(detail, "message: "); ok {
			configErr.message = after
		}
	}

	if payload != nil {
		if stringPayload, ok := payload.(*errorspb.StringPayload); ok {
			configErr.message = stringPayload.Msg
		}
	}

	return configErr
}
