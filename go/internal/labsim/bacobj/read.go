package bacobj

import (
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
	"github.com/angeles725/niagara-tools/go/internal/labsim/model"
)

// tableFor is the property table of a resolved object.
func tableFor(obj *object) []property {
	if obj == nil {
		return deviceTable
	}
	return obj.table
}

// lookup finds a property in a table.
func lookup(table []property, id bacnet.Property) *property {
	for i := range table {
		if table[i].id == id {
			return &table[i]
		}
	}
	return nil
}

// element applies an array index to a property's whole value: none gives all
// of it, 0 the length of an array, n its n-th element.
func element(p *property, all []bacnet.Value, index *uint32) ([]bacnet.Value, *bacnet.Error) {
	switch {
	case index == nil:
		return all, nil
	case !p.array:
		return nil, bacnet.ErrNotAnArray
	case *index == 0:
		return one(bacnet.Unsigned(uint32(len(all)))), nil
	case int(*index) > len(all):
		return nil, bacnet.ErrInvalidArrayIndex
	}
	return all[*index-1 : *index], nil
}

// Read is the value of one property at elapsed: one value for a scalar or an
// array element, every element for a whole array, none for an empty list.
func (d *Device) Read(id bacnet.ObjectID, ref bacnet.PropertyRef, elapsed time.Duration) ([]bacnet.Value, *bacnet.Error) {
	obj, _, ok := d.resolve(id)
	if !ok {
		return nil, bacnet.ErrUnknownObject
	}
	p := lookup(tableFor(obj), ref.Property)
	if p == nil {
		return nil, bacnet.ErrUnknownProperty
	}
	var all []bacnet.Value
	d.points.Do(elapsed, func(tx model.Tx) { all = p.get(reading{d: d, o: obj, tx: tx}) })
	return element(p, all, ref.Index)
}

// ReadSet reads the properties one of ALL, REQUIRED and OPTIONAL stands for,
// all at the same instant. An object the device has not gives one result: the
// error, under the special property itself.
func (d *Device) ReadSet(id bacnet.ObjectID, which bacnet.Property, elapsed time.Duration) []bacnet.ReadResult {
	obj, _, ok := d.resolve(id)
	if !ok {
		return []bacnet.ReadResult{{Ref: bacnet.PropertyRef{Property: which}, Err: bacnet.ErrUnknownObject}}
	}
	var out []bacnet.ReadResult
	d.points.Do(elapsed, func(tx model.Tx) {
		for _, p := range tableFor(obj) {
			if (which == bacnet.PropRequired && !p.required) || (which == bacnet.PropOptional && p.required) {
				continue
			}
			out = append(out, bacnet.ReadResult{Ref: bacnet.PropertyRef{Property: p.id}, Values: p.get(reading{d: d, o: obj, tx: tx})})
		}
	})
	return out
}

// IsSet reports whether a property identifier stands for a set of properties.
func IsSet(p bacnet.Property) bool {
	return p == bacnet.PropAll || p == bacnet.PropRequired || p == bacnet.PropOptional
}
