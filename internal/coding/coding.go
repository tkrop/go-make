// Package coding provides functions for encoding and decoding of objects to
// and from strings and byte slices. The encoding and decoding is done using
// JSON and YAML encoders and decoders. Instead of throwing errors, the
// functions return textually encoded errors for further analysis.
package coding

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Type encoding type.
type Type string

// Check whether yaml input should return a nil object.
var yamlNilObjectMatch = regexp.MustCompile("null\n?")

const (
	// TypeUnkown constant for unknown encoding type.
	TypeUnkown Type = "UNKNOWN"
	// TypeJSON constant for JSON encoding type.
	TypeJSON Type = "JSON"
	// TypeYAML constant for YAML encoding type.
	TypeYAML Type = "YAML"
)

// ToString encodes any object to a string using the requested encoder. If the
// encoding fails or the encoder is unknown, no error is returned but a failure
// report encoded in a string is returned for analysis.
func ToString(ctype Type, obj any) string {
	return string(ToBytes(ctype, obj))
}

// FromString converts a string into any object using the requested decoder.
// If the decoding fails or the decoder is unknown, no error is thrown but an
// error is returned for further analysis.
func FromString(ctype Type, s string, obj any) any {
	return FromBytes(ctype, []byte(s), obj)
}

// ToBytes encodes any object to a byte slice using the requested encoder. If
// the encoding fails or the encoder is unknown, no error is returned, but a
// failure report encoded in a byte slice is returned for analysis.
func ToBytes(ctype Type, obj any) []byte {
	switch ctype {
	case TypeJSON:
		if b, err := json.Marshal(obj); err != nil {
			return []byte(NewErrEncoding(obj, err).Error())
		} else {
			return b
		}
	case TypeYAML:
		if b, err := yaml.Marshal(obj); err != nil {
			return []byte(NewErrEncoding(obj, err).Error())
		} else {
			return b
		}
	case TypeUnkown:
		fallthrough
	default:
		return []byte(NewErrEncoding(obj, NewErrCoding(ctype)).Error())
	}
}

// FromBytes converts a byte slice into any object using the requested decoder.
// If the decoding fails or the decoder is unknown, no error is thrown but an
// error is returned for further analysis.
func FromBytes(ctype Type, b []byte, obj any) any {
	switch ctype {
	case TypeJSON:
		if err := json.Unmarshal(b, &obj); err != nil {
			return NewErrDecoding(string(b), err)
		}
		return obj
	case TypeYAML:
		if yamlNilObjectMatch.Match(b) {
			obj = nil
			return obj
		} else if err := yaml.Unmarshal(b, obj); err != nil {
			return NewErrDecoding(string(b), err)
		}
		return obj
	case TypeUnkown:
		fallthrough
	default:
		return NewErrDecoding(string(b), NewErrCoding(ctype))
	}
}

// ErrCoding is the error type for unknown encoding types.
var ErrCoding = errors.New("unknown coding")

// NewErrCoding creates a new unknown coding error with given encoding type.
func NewErrCoding(ctype Type) error {
	return fmt.Errorf("%w: %v", ErrCoding, ctype)
}

// ErrDecoding is the error type for decoding errors.
var ErrDecoding = errors.New("error decoding")

// NewErrDecoding creates a new decoding error with given string and given
// root cause decoding error.
func NewErrDecoding(buf string, err error) error {
	return fmt.Errorf("%w [%s]: %w", ErrDecoding, buf, err)
}

// ErrEncoding is the error type for encoding errors.
var ErrEncoding = errors.New("error encoding")

// NewErrEncoding creates a new encoding error with the given object to be
// encoded and given root cause encoding error.
func NewErrEncoding(obj any, err error) error {
	return fmt.Errorf("%w [%T]: %w", ErrEncoding, obj, err)
}
