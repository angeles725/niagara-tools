package labsim

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
	"github.com/angeles725/niagara-tools/go/internal/labsim/bacobj"
)

// probeError is the class and code of an error answer.
type probeError struct {
	Class string `json:"class"`
	Code  string `json:"code"`
}

// probeValue is one value with its application data type.
type probeValue struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// probeRow is one property of a ReadPropertyMultiple acknowledgement or of
// a COV notification.
type probeRow struct {
	Property string       `json:"property"`
	Index    *uint32      `json:"index,omitempty"`
	Values   []probeValue `json:"values,omitempty"`
	Error    *probeError  `json:"error,omitempty"`
}

// probeLine is one line bacnet-probe prints; each action fills its fields.
type probeLine struct {
	Service      string      `json:"service,omitempty"`
	From         string      `json:"from,omitempty"`
	Object       string      `json:"object,omitempty"`
	Device       string      `json:"device,omitempty"`
	Property     string      `json:"property,omitempty"`
	Index        *uint32     `json:"index,omitempty"`
	Priority     *uint32     `json:"priority,omitempty"`
	Written      string      `json:"written,omitempty"`
	Subscribed   string      `json:"subscribed,omitempty"`
	Process      *uint32     `json:"process,omitempty"`
	Lifetime     *uint32     `json:"lifetime,omitempty"`
	Remaining    *uint32     `json:"remaining,omitempty"`
	MaxAPDU      *uint32     `json:"max_apdu,omitempty"`
	Segmentation string      `json:"segmentation,omitempty"`
	Vendor       *uint32     `json:"vendor,omitempty"`
	Values       any         `json:"values,omitempty"`
	Results      []probeRow  `json:"results,omitempty"`
	Result       string      `json:"result,omitempty"`
	Error        *probeError `json:"error,omitempty"`
	Reject       string      `json:"reject,omitempty"`
	Abort        string      `json:"abort,omitempty"`
}

// rendered turns values into what a line prints; never nil, so an empty list
// prints as one.
func rendered(values []bacnet.Value) []probeValue {
	out := make([]probeValue, 0, len(values))
	for _, v := range values {
		out = append(out, probeValue{Type: v.Kind.String(), Value: v.String()})
	}
	return out
}

// parseObject reads "type,instance".
func parseObject(text string) (bacnet.ObjectID, error) {
	name, number, _ := strings.Cut(text, ",")
	objectType, known := bacnet.ParseObjectType(name)
	instance, err := strconv.ParseUint(number, 10, 32)
	if !known || err != nil || instance > bacnet.WildcardInstance {
		return bacnet.ObjectID{}, fmt.Errorf(msgProbeObject, text, strings.Join(bacnet.ObjectTypeNames(), ", "))
	}
	return bacnet.ObjectID{Type: objectType, Instance: uint32(instance)}, nil
}

// parseProperty reads a property name; the three that stand for a set are
// accepted only where sets is true.
func parseProperty(text string, sets bool) (bacnet.Property, error) {
	property, known := bacnet.ParseProperty(text)
	if !known || (!sets && bacobj.IsSet(property)) {
		return 0, fmt.Errorf(msgProbeProperty, text)
	}
	return property, nil
}

// parseIndex reads an array index.
func parseIndex(text string) (*uint32, error) {
	n, err := strconv.ParseUint(text, 10, 32)
	if err != nil {
		return nil, fmt.Errorf(msgProbeIndex, text)
	}
	index := uint32(n)
	return &index, nil
}

// parseValue reads the value to write to the present-value of object: its
// object type gives the BACnet type, and null relinquishes.
func parseValue(object bacnet.ObjectID, text string) (bacnet.Value, error) {
	kind, point := bacobj.KindOf(object.Type)
	if !point {
		return bacnet.Value{}, fmt.Errorf(msgProbePointType)
	}
	if text == "null" {
		return bacnet.Null(), nil
	}
	switch kind {
	case bacobj.Binary:
		if text == "0" || text == "1" {
			return bacnet.Enum(uint32(text[0] - '0')), nil
		}
		return bacnet.Value{}, fmt.Errorf(msgProbeValue, text, object, wantBinary)
	case bacobj.MultiState:
		if n, err := strconv.ParseUint(text, 10, 32); err == nil && n >= 1 {
			return bacnet.Unsigned(uint32(n)), nil
		}
		return bacnet.Value{}, fmt.Errorf(msgProbeValue, text, object, wantMultiState)
	}
	f, err := strconv.ParseFloat(text, 32)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return bacnet.Value{}, fmt.Errorf(msgProbeValue, text, object, wantAnalog)
	}
	return bacnet.Real(float32(f)), nil
}
