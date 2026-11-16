package bacnet

// The confirmed services (BACnetConfirmedServiceChoice).
const (
	ServiceConfirmedCOVNotification = 1
	ServiceSubscribeCOV             = 5
	ServiceReadProperty             = 12
	ServiceReadPropertyMultiple     = 14
	ServiceWriteProperty            = 15
)

// The unconfirmed services (BACnetUnconfirmedServiceChoice).
const (
	ServiceIAm                        = 0
	ServiceIHave                      = 1
	ServiceUnconfirmedCOVNotification = 2
	ServiceTimeSynchronization        = 6
	ServiceWhoHas                     = 7
	ServiceWhoIs                      = 8
	ServiceUTCTimeSynchronization     = 9
)

var confirmedNames = map[byte]string{
	0: "acknowledge-alarm", ServiceConfirmedCOVNotification: "confirmed-cov-notification", 2: "confirmed-event-notification",
	3: "get-alarm-summary", 4: "get-enrollment-summary", ServiceSubscribeCOV: "subscribe-cov", 6: "atomic-read-file",
	7: "atomic-write-file", 8: "add-list-element", 9: "remove-list-element", 10: "create-object", 11: "delete-object",
	ServiceReadProperty: "read-property", ServiceReadPropertyMultiple: "read-property-multiple",
	ServiceWriteProperty: "write-property", 16: "write-property-multiple", 17: "device-communication-control",
	18: "confirmed-private-transfer", 19: "confirmed-text-message", 20: "reinitialize-device", 26: "read-range",
	28: "subscribe-cov-property", 29: "get-event-information",
}

var unconfirmedNames = map[byte]string{
	ServiceIAm: "i-am", ServiceIHave: "i-have", ServiceUnconfirmedCOVNotification: "unconfirmed-cov-notification",
	3: "unconfirmed-event-notification", 4: "unconfirmed-private-transfer", 5: "unconfirmed-text-message",
	ServiceTimeSynchronization: "time-synchronization", ServiceWhoHas: "who-has", ServiceWhoIs: "who-is",
	ServiceUTCTimeSynchronization: "utc-time-synchronization",
}

// ServiceName names a service choice of a confirmed or an unconfirmed PDU.
func ServiceName(confirmed bool, service byte) string {
	if confirmed {
		return name(confirmedNames, service, "confirmed-service")
	}
	return name(unconfirmedNames, service, "unconfirmed-service")
}

// The bit of each service in a protocol-services-supported bit string
// (BACnetServicesSupported), which is not its service choice.
const (
	SupportedConfirmedCOVNotification   = 1
	SupportedSubscribeCOV               = 5
	SupportedReadProperty               = 12
	SupportedReadPropertyMultiple       = 14
	SupportedWriteProperty              = 15
	SupportedIAm                        = 26
	SupportedIHave                      = 27
	SupportedUnconfirmedCOVNotification = 28
	SupportedTimeSynchronization        = 32
	SupportedWhoHas                     = 33
	SupportedWhoIs                      = 34
	SupportedUTCTimeSynchronization     = 36
	ServicesSupportedBits               = 40 // the length the simulator gives the bit string
	ObjectTypesSupportedBits            = 25 // the same, of protocol-object-types-supported
	ProtocolVersion                     = 1
	ProtocolRevision                    = 4
)
