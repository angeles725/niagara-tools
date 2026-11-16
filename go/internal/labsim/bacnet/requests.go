package bacnet

// PropertyRef names a property and, for one element of an array, its index.
type PropertyRef struct {
	Property Property
	Index    *uint32
}

// append writes the reference with the context tags first and first+1.
func (p PropertyRef) append(b []byte, first byte) []byte {
	b = AppendCtxUnsigned(b, first, uint32(p.Property))
	if p.Index != nil {
		b = AppendCtxUnsigned(b, first+1, *p.Index)
	}
	return b
}

// ref reads a reference with the context tags first and first+1.
func (r *Reader) ref(first byte) (PropertyRef, error) {
	id, err := r.CtxUnsigned(first)
	if err != nil {
		return PropertyRef{}, err
	}
	p := PropertyRef{Property: Property(id)}
	if index, ok, err := r.OptCtxUnsigned(first + 1); err != nil {
		return p, err
	} else if ok {
		p.Index = &index
	}
	return p, nil
}

// end refuses data after a complete request.
func (r *Reader) end() error {
	if !r.Done() {
		return RejectTooManyArguments
	}
	return nil
}

// ReadProperty is the ReadProperty-Request.
type ReadProperty struct {
	Object ObjectID
	Ref    PropertyRef
}

func (q ReadProperty) Bytes() []byte { return q.Ref.append(AppendCtxOID(nil, 0, q.Object), 1) }

func DecodeReadProperty(data []byte) (q ReadProperty, err error) {
	r := NewReader(data)
	if q.Object, err = r.CtxOID(0); err != nil {
		return q, err
	}
	if q.Ref, err = r.ref(1); err != nil {
		return q, err
	}
	return q, r.end()
}

// ReadSpec is the properties wanted of one object.
type ReadSpec struct {
	Object ObjectID
	Refs   []PropertyRef
}

// ReadPropertyMultiple is the ReadPropertyMultiple-Request.
type ReadPropertyMultiple []ReadSpec

func (q ReadPropertyMultiple) Bytes() []byte {
	var b []byte
	for _, spec := range q {
		b = AppendOpening(AppendCtxOID(b, 0, spec.Object), 1)
		for _, ref := range spec.Refs {
			b = ref.append(b, 0)
		}
		b = AppendClosing(b, 1)
	}
	return b
}

func DecodeReadPropertyMultiple(data []byte) (ReadPropertyMultiple, error) {
	r := NewReader(data)
	var q ReadPropertyMultiple
	for len(q) == 0 || !r.Done() {
		object, err := r.CtxOID(0)
		if err != nil {
			return nil, err
		}
		if err := r.Open(1); err != nil {
			return nil, err
		}
		spec := ReadSpec{Object: object}
		for len(spec.Refs) == 0 || !r.AtClosing(1) {
			if r.AtClosing(1) {
				return nil, RejectMissingParameter
			}
			ref, err := r.ref(0)
			if err != nil {
				return nil, err
			}
			spec.Refs = append(spec.Refs, ref)
		}
		if err := r.Close(1); err != nil {
			return nil, err
		}
		q = append(q, spec)
	}
	return q, nil
}

// WriteProperty is the WriteProperty-Request.
type WriteProperty struct {
	Object      ObjectID
	Ref         PropertyRef
	Values      []Value
	HasPriority bool
	Priority    uint32
}

func (q WriteProperty) Bytes() []byte {
	b := AppendEnclosed(q.Ref.append(AppendCtxOID(nil, 0, q.Object), 1), 3, q.Values)
	if q.HasPriority {
		b = AppendCtxUnsigned(b, 4, q.Priority)
	}
	return b
}

func DecodeWriteProperty(data []byte) (q WriteProperty, err error) {
	r := NewReader(data)
	if q.Object, err = r.CtxOID(0); err != nil {
		return q, err
	}
	if q.Ref, err = r.ref(1); err != nil {
		return q, err
	}
	if q.Values, err = r.Enclosed(3); err != nil {
		return q, err
	}
	if q.Priority, q.HasPriority, err = r.OptCtxUnsigned(4); err != nil {
		return q, err
	}
	return q, r.end()
}

// SubscribeCOV is the SubscribeCOV-Request. With neither the confirmed flag
// nor a lifetime it cancels the subscription.
type SubscribeCOV struct {
	Process   uint32
	Object    ObjectID
	Cancel    bool
	Confirmed bool
	Lifetime  uint32 // seconds; 0 has no end
}

func (q SubscribeCOV) Bytes() []byte {
	b := AppendCtxOID(AppendCtxUnsigned(nil, 0, q.Process), 1, q.Object)
	if q.Cancel {
		return b
	}
	return AppendCtxUnsigned(AppendCtxBool(b, 2, q.Confirmed), 3, q.Lifetime)
}

func DecodeSubscribeCOV(data []byte) (q SubscribeCOV, err error) {
	r := NewReader(data)
	if q.Process, err = r.CtxUnsigned(0); err != nil {
		return q, err
	}
	if q.Object, err = r.CtxOID(1); err != nil {
		return q, err
	}
	flagged := r.AtContext(2)
	if flagged {
		if q.Confirmed, err = r.CtxBool(2); err != nil {
			return q, err
		}
	}
	lifetime, timed, err := r.OptCtxUnsigned(3)
	if err != nil {
		return q, err
	}
	q.Lifetime, q.Cancel = lifetime, !flagged && !timed
	return q, r.end()
}
