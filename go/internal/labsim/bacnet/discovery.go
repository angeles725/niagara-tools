package bacnet

// WhoIs is the Who-Is-Request: every device, or those whose instance is in a
// range.
type WhoIs struct {
	Ranged    bool
	Low, High uint32
}

func (q WhoIs) Bytes() []byte {
	if !q.Ranged {
		return nil
	}
	return AppendCtxUnsigned(AppendCtxUnsigned(nil, 0, q.Low), 1, q.High)
}

// limits reads the optional instance range Who-Is and Who-Has start with.
func (r *Reader) limits() (ranged bool, low, high uint32, err error) {
	if low, ranged, err = r.OptCtxUnsigned(0); err != nil || !ranged {
		return false, 0, 0, err
	}
	high, err = r.CtxUnsigned(1)
	return true, low, high, err
}

func DecodeWhoIs(data []byte) (q WhoIs, err error) {
	r := NewReader(data)
	if q.Ranged, q.Low, q.High, err = r.limits(); err != nil {
		return q, err
	}
	return q, r.end()
}

// Matches reports whether a device instance is asked for.
func (q WhoIs) Matches(instance uint32) bool {
	return !q.Ranged || (instance >= q.Low && instance <= q.High)
}

// IAm is the I-Am-Request.
type IAm struct {
	Device       ObjectID
	MaxAPDU      uint32
	Segmentation uint32
	Vendor       uint32
}

func (m IAm) Bytes() []byte {
	b := Unsigned(m.MaxAPDU).Append(OID(m.Device).Append(nil))
	return Unsigned(m.Vendor).Append(Enum(m.Segmentation).Append(b))
}

// sequence reads application values of the given kinds, in order.
func (r *Reader) sequence(kinds ...Kind) ([]Value, error) {
	out := make([]Value, 0, len(kinds))
	for _, kind := range kinds {
		v, err := r.Value()
		if err != nil {
			return nil, err
		}
		if v.Kind != kind {
			return nil, RejectInvalidParameterType
		}
		out = append(out, v)
	}
	return out, r.end()
}

func DecodeIAm(data []byte) (IAm, error) {
	v, err := NewReader(data).sequence(KindOID, KindUnsigned, KindEnum, KindUnsigned)
	if err != nil {
		return IAm{}, err
	}
	return IAm{Device: v[0].Object, MaxAPDU: v[1].Uint, Segmentation: v[2].Uint, Vendor: v[3].Uint}, nil
}

// WhoHas is the Who-Has-Request: which device has an object, named or
// identified.
type WhoHas struct {
	Ranged    bool
	Low, High uint32
	ByName    bool
	Object    ObjectID
	Name      string
}

func (q WhoHas) Bytes() []byte {
	b := WhoIs{Ranged: q.Ranged, Low: q.Low, High: q.High}.Bytes()
	if q.ByName {
		return AppendCtxText(b, 3, q.Name)
	}
	return AppendCtxOID(b, 2, q.Object)
}

func DecodeWhoHas(data []byte) (q WhoHas, err error) {
	r := NewReader(data)
	if q.Ranged, q.Low, q.High, err = r.limits(); err != nil {
		return q, err
	}
	if q.ByName = r.AtContext(3); q.ByName {
		q.Name, err = r.CtxText(3)
	} else {
		q.Object, err = r.CtxOID(2)
	}
	if err != nil {
		return q, err
	}
	return q, r.end()
}

// IHave is the I-Have-Request.
type IHave struct {
	Device ObjectID
	Object ObjectID
	Name   string
}

func (m IHave) Bytes() []byte {
	return Text(m.Name).Append(OID(m.Object).Append(OID(m.Device).Append(nil)))
}

func DecodeIHave(data []byte) (IHave, error) {
	v, err := NewReader(data).sequence(KindOID, KindOID, KindString)
	if err != nil {
		return IHave{}, err
	}
	return IHave{Device: v[0].Object, Object: v[1].Object, Name: v[2].Text}, nil
}
