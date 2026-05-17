package validate

import (
	"context"
	"fmt"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/errors/errbase"
	"github.com/cockroachdb/errors/errorspb"
	"github.com/gogo/protobuf/proto"
)

type ValidationError struct {
	field string
	value any
	rule  string
}

var _ error = (*ValidationError)(nil)
var _ fmt.Formatter = (*ValidationError)(nil)
var _ errbase.SafeFormatter = (*ValidationError)(nil)

func ValidationFailedf(field string, value any, rule string) error {
	return &ValidationError{field: field, value: value, rule: rule}
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: field '%s' failed rule '%s'", e.field, e.rule)
}

func (e *ValidationError) SafeDetails() []string {
	return []string{
		fmt.Sprintf("field: %s", e.field),
		fmt.Sprintf("rule: %s", e.rule),
	}
}

func (e *ValidationError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "validationError{field: %q, value: %v, rule: %q}", e.field, e.value, e.rule)
			return
		}
		fallthrough
	case 's', 'q':
		fmt.Fprint(s, e.Error())
	}
}

func (e *ValidationError) SafeFormatError(p errbase.Printer) (next error) {
	if p.Detail() {
		// value may be PII; print it only in detailed output
		p.Printf("; value=%v", e.value)
	}
	return nil
}

func init() {
	errors.RegisterLeafEncoder(errbase.GetTypeKey((*ValidationError)(nil)), encodeValidationError)
	errors.RegisterLeafDecoder(errbase.GetTypeKey((*ValidationError)(nil)), decodeValidationError)
}

func encodeValidationError(ctx context.Context, err error) (msg string, safeDetails []string, payload proto.Message) {
	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return validationErr.Error(), validationErr.SafeDetails(), &errorspb.StringPayload{Msg: fmt.Sprintf("%s", validationErr.value)}
	}

	return err.Error(), nil, nil
}

func decodeValidationError(ctx context.Context, msg string, safeDetails []string, payload proto.Message) error {
	validationErr := new(ValidationError)
	for _, detail := range safeDetails {
		if after, ok := strings.CutPrefix(detail, "field: "); ok {
			validationErr.field = after
		}
		if after, ok := strings.CutPrefix(detail, "rule: "); ok {
			validationErr.rule = after
		}
	}

	if payload != nil {
		if stringPayload, ok := payload.(*errorspb.StringPayload); ok {
			validationErr.value = stringPayload.Msg
		}
	}

	return validationErr
}
