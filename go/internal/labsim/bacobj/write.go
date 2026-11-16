package bacobj

import (
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
	"github.com/angeles725/niagara-tools/go/internal/labsim/model"
)

// defaultPriority is the priority of a command that states none.
const defaultPriority = Priorities

// Write applies a WriteProperty request and returns nil, or the error the
// device answers. The checks run in the order a device makes them: is the
// object there, has it the property, may the property be written, is the
// value of its type and in its range.
func (d *Device) Write(q bacnet.WriteProperty, elapsed time.Duration) *bacnet.Error {
	obj, isDevice, ok := d.resolve(q.Object)
	if !ok {
		return bacnet.ErrUnknownObject
	}
	if lookup(tableFor(obj), q.Ref.Property) == nil {
		return bacnet.ErrUnknownProperty
	}
	present := q.Ref.Property == bacnet.PropPresentValue
	relinquish := q.Ref.Property == bacnet.PropRelinquishDefault
	if isDevice || !(present || relinquish) || (present && !obj.Commandable && !obj.Writable) {
		return bacnet.ErrWriteAccessDenied
	}
	if q.Ref.Index != nil {
		return bacnet.ErrNotAnArray
	}
	if len(q.Values) != 1 {
		return bacnet.ErrInvalidDataType
	}
	value := q.Values[0]
	if present && obj.Commandable {
		return d.command(obj, q, value, elapsed)
	}
	v, err := obj.accept(value)
	if err != nil {
		return err
	}
	// A plain write, and a write of relinquish-default, change the field value.
	d.points.Do(elapsed, func(tx model.Tx) { tx.Set(obj.Point, v) })
	return nil
}

// command writes or relinquishes one slot of the priority array and shows the
// winning command through the model.
func (d *Device) command(obj *object, q bacnet.WriteProperty, value bacnet.Value, elapsed time.Duration) *bacnet.Error {
	priority := uint32(defaultPriority)
	if q.HasPriority {
		priority = q.Priority
	}
	if priority < 1 || priority > Priorities {
		return bacnet.ErrValueOutOfRange
	}
	var slot *float64
	if !value.IsNull() {
		v, err := obj.accept(value)
		if err != nil {
			return err
		}
		slot = &v
	}
	d.points.Do(elapsed, func(tx model.Tx) {
		obj.priority[priority-1] = slot
		var winner *float64
		for _, s := range obj.priority {
			if s != nil {
				winner = s
				break
			}
		}
		tx.Override(obj.Point, winner)
	})
	return nil
}
