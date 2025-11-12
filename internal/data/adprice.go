package data

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Price int32

func (p Price) MarshalJSON() ([]byte, error) {
	quotedJSONValue := strconv.Quote(fmt.Sprintf("%d $", p))
	return []byte(quotedJSONValue), nil
}

var ErrInvalidPriceFormat = errors.New("invalid price format")

func (p *Price) UnmarshalJSON(JSONValue []byte) error {
	unquotedJSONValue, err := strconv.Unquote(string(JSONValue))

	if err != nil {
		return ErrInvalidPriceFormat
	}

	parts := strings.Split(unquotedJSONValue, " ")
	if len(parts) != 2 || parts[1] != "$" {
		return ErrInvalidPriceFormat
	}

	i, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return ErrInvalidPriceFormat
	}

	*p = Price(i)

	return nil
}
