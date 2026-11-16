package bacserver

import "github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"

// network takes the NPDU of a frame to the application layer. The device is
// on one local network and routes nothing: a message for another network, and
// every network layer message (Who-Is-Router-To-Network among them), is
// ignored.
func (x *exchange) network(data []byte) {
	n, ok := bacnet.ParseNPDU(data)
	if !ok {
		x.ignore(reasonBadNPDU)
		return
	}
	x.npdu = n
	if n.Network {
		x.ignore(reasonNetworkLayer)
		return
	}
	if n.HasDest && n.DNet != bacnet.GlobalBroadcast {
		x.ignore(reasonOtherNetwork)
		return
	}
	a, ok := bacnet.ParseAPDU(n.APDU)
	if !ok {
		x.ignore(reasonBadAPDU)
		return
	}
	x.entry.PDU = bacnet.PDUName(a.Type)
	switch a.Type {
	case bacnet.PDUConfirmed:
		x.confirmed(a)
	case bacnet.PDUUnconfirmed:
		x.unconfirmed(a)
	default:
		// An acknowledgement of a confirmed COV notification, or a PDU that
		// answers nothing the simulator asked.
		x.ignore(reasonNotRequest)
	}
}

// unconfirmed serves an unconfirmed request; a service the simulator has not
// is ignored, for an unconfirmed request is never rejected.
func (x *exchange) unconfirmed(a bacnet.APDU) {
	x.entry.Service = bacnet.ServiceName(false, a.Service)
	if x.offline() {
		return
	}
	svc, known := unconfirmedServices[a.Service]
	if !known {
		x.ignore(reasonUnsupported)
		return
	}
	svc.handle(x, a.Data)
}
