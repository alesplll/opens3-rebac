package validate

import (
	"encoding/json"

	"github.com/pkg/errors"
)

type ValidationErrors struct {
	Messages []string `json:"error_messages"`
}

func (v *ValidationErrors) addError(message string) {
	v.Messages = append(v.Messages, message)
}

func NewValidationErrors(messages ...string) *ValidationErrors {
	return &ValidationErrors{
		Messages: messages,
	}
}

func (v *ValidationErrors) Error() string {
	data, err := json.Marshal(v.Messages)
	if err != nil {
		return err.Error()
	}

	return string(data)
}

func IsValidationError(err error) bool {
	var ve *ValidationErrors
	return errors.As(err, &ve)
}
