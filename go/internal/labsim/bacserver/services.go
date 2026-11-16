package bacserver

import (
	"fmt"
	"slices"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
	"github.com/angeles725/niagara-tools/go/internal/labsim/bacobj"
)

// describe writes the object and the property of a request into its log entry.
func (x *exchange) describe(object bacnet.ObjectID, ref bacnet.PropertyRef) {
	x.entry.Object, x.entry.Property = object.String(), ref.Property.String()
	if ref.Index != nil {
		index := int(*ref.Index)
		x.entry.Index = &index
	}
}

// touched lists the model points behind objects, each once.
func (x *exchange) touched(objects ...bacnet.ObjectID) []string {
	var points []string
	for _, id := range objects {
		if p := x.s.device.Point(id); p != "" && !slices.Contains(points, p) {
			points = append(points, p)
		}
	}
	return points
}

// texts renders values for the log, the first ones only.
func texts(values []bacnet.Value) []string {
	out := make([]string, 0, min(len(values), maxLoggedValues))
	for _, v := range values[:min(len(values), maxLoggedValues)] {
		out = append(out, v.String())
	}
	return out
}

// prepareRead decodes ReadProperty.
func prepareRead(x *exchange, data []byte) (call, error) {
	q, err := bacnet.DecodeReadProperty(data)
	if err != nil {
		return call{}, err
	}
	x.describe(q.Object, q.Ref)
	return call{points: x.touched(q.Object), read: true, run: func(x *exchange) ([]byte, *bacnet.Error) {
		values, fault := x.s.device.Read(q.Object, q.Ref, x.elapsed)
		if fault != nil {
			return nil, fault
		}
		x.entry.Read = texts(values)
		return bacnet.ReadPropertyAck{Object: q.Object, Ref: q.Ref, Values: values}.Bytes(), nil
	}}, nil
}

// prepareReadMultiple decodes ReadPropertyMultiple. Each reference is answered
// by itself, with its values or its error; ALL, REQUIRED and OPTIONAL are
// answered with the properties they stand for.
func prepareReadMultiple(x *exchange, data []byte) (call, error) {
	q, err := bacnet.DecodeReadPropertyMultiple(data)
	if err != nil {
		return call{}, err
	}
	objects, references := make([]bacnet.ObjectID, 0, len(q)), 0
	for _, spec := range q {
		objects = append(objects, spec.Object)
		references += len(spec.Refs)
	}
	x.entry.Detail = fmt.Sprintf("%d objects, %d references", len(q), references)
	if len(q) == 1 {
		x.entry.Object = q[0].Object.String()
	}
	return call{points: x.touched(objects...), read: true, run: func(x *exchange) ([]byte, *bacnet.Error) {
		device := x.s.device
		ack := make(bacnet.ReadPropertyMultipleAck, 0, len(q))
		for _, spec := range q {
			result := bacnet.ObjectResult{Object: spec.Object}
			for _, ref := range spec.Refs {
				if bacobj.IsSet(ref.Property) {
					result.Results = append(result.Results, device.ReadSet(spec.Object, ref.Property, x.elapsed)...)
					continue
				}
				values, fault := device.Read(spec.Object, ref, x.elapsed)
				result.Results = append(result.Results, bacnet.ReadResult{Ref: ref, Values: values, Err: fault})
			}
			ack = append(ack, result)
		}
		return ack.Bytes(), nil
	}}, nil
}

// prepareWrite decodes WriteProperty. A write that succeeds is followed by the
// COV notifications it causes.
func prepareWrite(x *exchange, data []byte) (call, error) {
	q, err := bacnet.DecodeWriteProperty(data)
	if err != nil {
		return call{}, err
	}
	x.describe(q.Object, q.Ref)
	if q.HasPriority {
		priority := int(q.Priority)
		x.entry.Priority = &priority
	}
	x.entry.Written = strings.Join(texts(q.Values), ",")
	return call{points: x.touched(q.Object), run: func(x *exchange) ([]byte, *bacnet.Error) {
		return nil, x.s.device.Write(q, x.elapsed)
	}, after: (*Server).CheckCOV}, nil
}
