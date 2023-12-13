package coding_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/tkrop/go-make/internal/coding"
	"github.com/tkrop/go-testing/test"
)

var (
	errDecode = fmt.Errorf("decode error")
	errEncode = fmt.Errorf("encode error")
)

type object struct {
	Type  string
	Value string
	Flag  bool
}

type delegate struct {
	Type  string `json:"type" yaml:"type"`
	Value string `json:"value,omitempty" yaml:"value,omitempty"`
	Flag  bool   `json:"flag" yaml:"flag"`
}

func (o *object) MarshalJSON() ([]byte, error) {
	if o == nil {
		return json.Marshal(nil)
	} else if o.Type == "encode" {
		return nil, errEncode
	}
	return json.Marshal(&delegate{
		Type:  o.Type,
		Value: o.Value,
		Flag:  o.Flag,
	})
}

func (o *object) UnmarshalJSON(b []byte) error {
	d := &delegate{}
	if err := json.Unmarshal(b, d); err != nil {
		return err
	} else if d.Type == "decode" {
		return errDecode
	}
	o.Type = d.Type
	o.Value = d.Value
	o.Flag = d.Flag
	return nil
}

func (o *object) MarshalYAML() (any, error) {
	node := &yaml.Node{}
	if o == nil {
		return node, nil
	} else if o.Type == "encode" {
		return nil, errEncode
	}
	err := node.Encode(&delegate{
		Type:  o.Type,
		Value: o.Value,
		Flag:  o.Flag,
	})
	return node, err
}

func (o *object) UnmarshalYAML(unmarshal func(any) error) error {
	d := &delegate{}
	if err := unmarshal(d); err != nil {
		return err
	} else if d.Type == "decode" {
		return errDecode
	}
	o.Type = d.Type
	o.Value = d.Value
	o.Flag = d.Flag
	return nil
}

var (
	errUnknownCodingString = coding.NewErrEncoding(nil,
		coding.NewErrCoding(coding.TypeUnkown)).Error()
	//nolint:errchkjson // used by test.
	_, errMarshalJSON = json.Marshal(&object{Type: "encode"})
	_, errMarshalYAML = yaml.Marshal(&object{Type: "encode"})
)

type StringParams struct {
	from         any
	to           any
	ctype        coding.Type
	expectString string
	expectResult any
}

var testStringParams = map[string]StringParams{
	// Type UNKNOWN tests.
	"unknown nil object": {
		ctype:        coding.TypeUnkown,
		to:           &object{},
		expectString: errUnknownCodingString,
		expectResult: coding.NewErrDecoding(errUnknownCodingString,
			coding.NewErrCoding(coding.TypeUnkown)),
	},

	// Type JSON tests.
	"json nil object": {
		ctype:        coding.TypeJSON,
		to:           &object{},
		expectString: `null`,
	},
	"json empty object": {
		ctype:        coding.TypeJSON,
		from:         &object{},
		to:           &object{},
		expectString: `{"type":"","flag":false}`,
		expectResult: &object{},
	},
	"json full token": {
		ctype:        coding.TypeJSON,
		from:         &object{Type: "object", Value: "string", Flag: true},
		to:           &object{},
		expectString: `{"type":"object","value":"string","flag":true}`,
		expectResult: &object{Type: "object", Value: "string", Flag: true},
	},
	"json encode failure": {
		ctype:        coding.TypeJSON,
		from:         &object{Type: "encode"},
		expectString: coding.NewErrEncoding(&object{}, errMarshalJSON).Error(),
	},
	"json decode failure": {
		ctype:        coding.TypeJSON,
		from:         &object{Type: "decode"},
		to:           &object{},
		expectString: `{"type":"decode","flag":false}`,
		expectResult: coding.NewErrDecoding(
			`{"type":"decode","flag":false}`, errDecode),
	},

	// Type YAML tests.
	"yaml nil object": {
		ctype:        coding.TypeYAML,
		to:           &object{},
		expectString: "null\n",
	},
	"yaml empty object": {
		ctype:        coding.TypeYAML,
		from:         &object{},
		to:           &object{},
		expectString: "type: \"\"\nflag: false\n",
		expectResult: &object{},
	},
	"yaml full object": {
		ctype:        coding.TypeYAML,
		from:         &object{Type: "object", Value: "string", Flag: true},
		to:           &object{},
		expectString: "type: object\nvalue: string\nflag: true\n",
		expectResult: &object{Type: "object", Value: "string", Flag: true},
	},
	"yaml encode failure": {
		ctype:        coding.TypeYAML,
		from:         &object{Type: "encode"},
		expectString: coding.NewErrEncoding(&object{}, errMarshalYAML).Error(),
	},
	"yaml decode failure": {
		ctype:        coding.TypeYAML,
		from:         &object{Type: "decode"},
		to:           &object{},
		expectString: "type: decode\nflag: false\n",
		expectResult: coding.NewErrDecoding(
			"type: decode\nflag: false\n", errDecode),
	},
}

func TestToFromString(t *testing.T) {
	test.Map(t, testStringParams).
		Run(func(t test.Test, param StringParams) {
			// Given
			object := param.from

			// When
			str := coding.ToString(param.ctype, object)

			// Then
			assert.Equal(t, param.expectString, str)

			if param.to != nil {
				// When
				result := coding.FromString(param.ctype, str, param.to)

				// Then
				assert.Equal(t, param.expectResult, result)
			}
		})
}
