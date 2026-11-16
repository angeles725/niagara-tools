package labsim

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
)

// The modes of a subscription.
const (
	modeUnconfirmed = "unconfirmed"
	modeConfirmed   = "confirmed"
	modeCancel      = "cancel"
	textCancelled   = "cancelled"
)

func prepareProbeSubscribe(p parsed, args []string) (probeRun, error) {
	object, err := parseObject(args[0])
	if err != nil {
		return nil, err
	}
	mode := p.option("mode", modeUnconfirmed)
	if mode != modeUnconfirmed && mode != modeConfirmed && mode != modeCancel {
		return nil, fmt.Errorf(msgProbeMode, mode)
	}
	process, err1 := whole(p, "process", "1", 0, 4194303)
	lifetime, err2 := whole(p, "lifetime", "60", 0, 86400)
	count, err3 := whole(p, "count", "1", 1, 1000)
	for _, err := range []error{err1, err2, err3} {
		if err != nil {
			return nil, err
		}
	}
	q := bacnet.SubscribeCOV{Process: uint32(process), Object: object, Cancel: mode == modeCancel, Confirmed: mode == modeConfirmed, Lifetime: uint32(lifetime)}
	line := probeLine{Object: object.String(), Subscribed: mode, Process: &q.Process, Lifetime: &q.Lifetime}
	if q.Cancel {
		line.Subscribed, line.Lifetime = textCancelled, nil
	}
	return func(c *probeClient) (int, error) {
		a, err := c.confirmed(bacnet.ServiceSubscribeCOV, q.Bytes())
		if err != nil {
			return exitFailed, err
		}
		if code, err := c.finish(line, a); code != exitOK || q.Cancel {
			return code, err
		}
		for got := 0; got < count; {
			a, err := c.next()
			if err != nil {
				return exitFailed, err
			}
			n, err := bacnet.DecodeCOVNotification(a.Data)
			confirmed := a.Type == bacnet.PDUConfirmed && a.Service == bacnet.ServiceConfirmedCOVNotification
			if err != nil || !(confirmed || (a.Type == bacnet.PDUUnconfirmed && a.Service == bacnet.ServiceUnconfirmedCOVNotification)) {
				continue
			}
			if confirmed {
				if err := c.send(bacnet.SimpleACK(a.InvokeID, a.Service), false); err != nil {
					return exitFailed, err
				}
			}
			c.print(notificationLine(n, confirmed))
			got++
		}
		return exitOK, nil
	}, nil
}

// notificationLine renders a COV notification.
func notificationLine(n bacnet.COVNotification, confirmed bool) probeLine {
	service := bacnet.ServiceUnconfirmedCOVNotification
	if confirmed {
		service = bacnet.ServiceConfirmedCOVNotification
	}
	rows := make([]probeRow, 0, len(n.Values))
	for _, v := range n.Values {
		rows = append(rows, probeRow{Property: v.Ref.Property.String(), Index: v.Ref.Index, Values: rendered(v.Values)})
	}
	return probeLine{Service: bacnet.ServiceName(confirmed, byte(service)), Object: n.Object.String(), Device: n.Device.String(),
		Process: &n.Process, Remaining: &n.Remaining, Values: rows}
}
