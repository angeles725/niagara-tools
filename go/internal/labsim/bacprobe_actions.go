package labsim

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
)

// textNone is how the probe prints no-segmentation.
const textNone = "none"

type probeRun = func(c *probeClient) (int, error)

func prepareWhoIs(_ parsed, args []string) (probeRun, error) {
	q := bacnet.WhoIs{}
	if len(args) == 1 {
		return nil, fmt.Errorf(msgProbeArgs, "who-is", argsWhoIs)
	}
	if len(args) == 2 {
		low, err1 := strconv.ParseUint(args[0], 10, 32)
		high, err2 := strconv.ParseUint(args[1], 10, 32)
		if err1 != nil || err2 != nil || low > high || high > bacnet.WildcardInstance {
			return nil, fmt.Errorf(msgProbeRange, args[0], args[1])
		}
		q = bacnet.WhoIs{Ranged: true, Low: uint32(low), High: uint32(high)}
	}
	return func(c *probeClient) (int, error) {
		if err := c.send(bacnet.UnconfirmedRequest(bacnet.ServiceWhoIs, q.Bytes()), false); err != nil {
			return exitFailed, err
		}
		for {
			a, err := c.next()
			if err != nil {
				return exitFailed, err
			}
			iam, err := bacnet.DecodeIAm(a.Data)
			if a.Type != bacnet.PDUUnconfirmed || a.Service != bacnet.ServiceIAm || err != nil {
				continue
			}
			segmentation := strconv.FormatUint(uint64(iam.Segmentation), 10)
			if iam.Segmentation == bacnet.SegmentationNone {
				segmentation = textNone
			}
			c.print(probeLine{Service: bacnet.ServiceName(false, bacnet.ServiceIAm), From: c.server.String(), Device: iam.Device.String(),
				MaxAPDU: &iam.MaxAPDU, Segmentation: segmentation, Vendor: &iam.Vendor})
			return exitOK, nil
		}
	}, nil
}

func prepareProbeRead(_ parsed, args []string) (probeRun, error) {
	object, err := parseObject(args[0])
	if err != nil {
		return nil, err
	}
	q := bacnet.ReadProperty{Object: object}
	if q.Ref.Property, err = parseProperty(args[1], false); err != nil {
		return nil, err
	}
	if len(args) == 3 {
		if q.Ref.Index, err = parseIndex(args[2]); err != nil {
			return nil, err
		}
	}
	return func(c *probeClient) (int, error) {
		a, err := c.confirmed(bacnet.ServiceReadProperty, q.Bytes())
		if err != nil {
			return exitFailed, err
		}
		line := probeLine{Object: object.String(), Property: q.Ref.Property.String(), Index: q.Ref.Index}
		if ack, err := bacnet.DecodeReadPropertyAck(a.Data); a.Type == bacnet.PDUComplexACK && err == nil {
			line.Values = rendered(ack.Values)
		}
		return c.finish(line, a)
	}, nil
}

func prepareProbeReadMultiple(_ parsed, args []string) (probeRun, error) {
	object, err := parseObject(args[0])
	if err != nil {
		return nil, err
	}
	spec := bacnet.ReadSpec{Object: object}
	for _, name := range strings.Split(args[1], ",") {
		property, err := parseProperty(name, true)
		if err != nil {
			return nil, err
		}
		spec.Refs = append(spec.Refs, bacnet.PropertyRef{Property: property})
	}
	return func(c *probeClient) (int, error) {
		a, err := c.confirmed(bacnet.ServiceReadPropertyMultiple, bacnet.ReadPropertyMultiple{spec}.Bytes())
		if err != nil {
			return exitFailed, err
		}
		line := probeLine{Object: object.String()}
		ack, err := bacnet.DecodeReadPropertyMultipleAck(a.Data)
		for _, result := range ack {
			if a.Type != bacnet.PDUComplexACK || err != nil {
				break
			}
			for _, r := range result.Results {
				row := probeRow{Property: r.Ref.Property.String(), Index: r.Ref.Index}
				if r.Err != nil {
					row.Error = &probeError{Class: r.Err.ClassName(), Code: r.Err.CodeName()}
				} else {
					row.Values = rendered(r.Values)
				}
				line.Results = append(line.Results, row)
			}
		}
		return c.finish(line, a)
	}, nil
}

func prepareProbeWrite(p parsed, args []string) (probeRun, error) {
	object, err := parseObject(args[0])
	if err != nil {
		return nil, err
	}
	value, err := parseValue(object, args[1])
	if err != nil {
		return nil, err
	}
	q := bacnet.WriteProperty{Object: object, Ref: bacnet.PropertyRef{Property: bacnet.PropPresentValue}, Values: []bacnet.Value{value}}
	line := probeLine{Object: object.String(), Property: q.Ref.Property.String(), Written: value.String(), Result: "ok"}
	if _, given := p.options["priority"]; given {
		priority, err := whole(p, "priority", "", 1, 16)
		if err != nil {
			return nil, err
		}
		q.HasPriority, q.Priority = true, uint32(priority)
		line.Priority = &q.Priority
	}
	return func(c *probeClient) (int, error) {
		a, err := c.confirmed(bacnet.ServiceWriteProperty, q.Bytes())
		if err != nil {
			return exitFailed, err
		}
		return c.finish(line, a)
	}, nil
}
