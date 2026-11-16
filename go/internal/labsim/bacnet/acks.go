package bacnet

// ReadPropertyAck is the ReadProperty-ACK: the values of a property, one for a
// scalar or an array element, several for a whole array or list.
type ReadPropertyAck struct {
	Object ObjectID
	Ref    PropertyRef
	Values []Value
}

func (a ReadPropertyAck) Bytes() []byte {
	return AppendEnclosed(a.Ref.append(AppendCtxOID(nil, 0, a.Object), 1), 3, a.Values)
}

func DecodeReadPropertyAck(data []byte) (a ReadPropertyAck, err error) {
	r := NewReader(data)
	if a.Object, err = r.CtxOID(0); err != nil {
		return a, err
	}
	if a.Ref, err = r.ref(1); err != nil {
		return a, err
	}
	if a.Values, err = r.Enclosed(3); err != nil {
		return a, err
	}
	return a, r.end()
}

// ReadResult is one property of a ReadPropertyMultiple-ACK: its values, or the
// error that reading it gave.
type ReadResult struct {
	Ref    PropertyRef
	Values []Value
	Err    *Error
}

// ObjectResult is the properties read of one object.
type ObjectResult struct {
	Object  ObjectID
	Results []ReadResult
}

// ReadPropertyMultipleAck is the ReadPropertyMultiple-ACK.
type ReadPropertyMultipleAck []ObjectResult

func (a ReadPropertyMultipleAck) Bytes() []byte {
	var b []byte
	for _, object := range a {
		b = AppendOpening(AppendCtxOID(b, 0, object.Object), 1)
		for _, res := range object.Results {
			b = res.Ref.append(b, 2)
			if res.Err != nil {
				b = AppendEnclosed(b, 5, []Value{Enum(res.Err.Class), Enum(res.Err.Code)})
			} else {
				b = AppendEnclosed(b, 4, res.Values)
			}
		}
		b = AppendClosing(b, 1)
	}
	return b
}

func DecodeReadPropertyMultipleAck(data []byte) (ReadPropertyMultipleAck, error) {
	r := NewReader(data)
	var a ReadPropertyMultipleAck
	for !r.Done() {
		object, err := r.CtxOID(0)
		if err != nil {
			return nil, err
		}
		if err := r.Open(1); err != nil {
			return nil, err
		}
		result := ObjectResult{Object: object}
		for !r.AtClosing(1) {
			res := ReadResult{}
			if res.Ref, err = r.ref(2); err != nil {
				return nil, err
			}
			if r.AtOpening(5) {
				pair, err := r.Enclosed(5)
				if err != nil || len(pair) != 2 {
					return nil, RejectInvalidTag
				}
				res.Err = &Error{Class: pair[0].Uint, Code: pair[1].Uint}
			} else if res.Values, err = r.Enclosed(4); err != nil {
				return nil, err
			}
			result.Results = append(result.Results, res)
		}
		if err := r.Close(1); err != nil {
			return nil, err
		}
		a = append(a, result)
	}
	return a, nil
}

// PropertyValue is one property and its values in a COV notification.
type PropertyValue struct {
	Ref    PropertyRef
	Values []Value
}

// COVNotification is the service data of a confirmed or an unconfirmed COV
// notification.
type COVNotification struct {
	Process   uint32
	Device    ObjectID
	Object    ObjectID
	Remaining uint32 // seconds the subscription still lasts; 0 has no end
	Values    []PropertyValue
}

func (n COVNotification) Bytes() []byte {
	b := AppendCtxOID(AppendCtxUnsigned(nil, 0, n.Process), 1, n.Device)
	b = AppendOpening(AppendCtxUnsigned(AppendCtxOID(b, 2, n.Object), 3, n.Remaining), 4)
	for _, v := range n.Values {
		b = AppendEnclosed(v.Ref.append(b, 0), 2, v.Values)
	}
	return AppendClosing(b, 4)
}

func DecodeCOVNotification(data []byte) (n COVNotification, err error) {
	r := NewReader(data)
	if n.Process, err = r.CtxUnsigned(0); err != nil {
		return n, err
	}
	if n.Device, err = r.CtxOID(1); err != nil {
		return n, err
	}
	if n.Object, err = r.CtxOID(2); err != nil {
		return n, err
	}
	if n.Remaining, err = r.CtxUnsigned(3); err != nil {
		return n, err
	}
	if err = r.Open(4); err != nil {
		return n, err
	}
	for !r.AtClosing(4) {
		v := PropertyValue{}
		if v.Ref, err = r.ref(0); err != nil {
			return n, err
		}
		if v.Values, err = r.Enclosed(2); err != nil {
			return n, err
		}
		if _, _, err = r.OptCtxUnsigned(3); err != nil { // a priority, which the simulator never sends
			return n, err
		}
		n.Values = append(n.Values, v)
	}
	return n, r.Close(4)
}
