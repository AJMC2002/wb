package kafkain

import (
	"bytes"
	"encoding/json"
	"fmt"

	"demo_service/internal/core/domain"
)

func DecodeOrder(b []byte) (domain.Order, error) {
	var o domain.Order

	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()

	if err := dec.Decode(&o); err != nil {
		return domain.Order{}, fmt.Errorf("json decode: %w", err)
	}

	if err := o.Validate(); err != nil {
		return domain.Order{}, fmt.Errorf("domain validate: %w", err)
	}

	return o, nil
}
