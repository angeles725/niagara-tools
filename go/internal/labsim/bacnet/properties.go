package bacnet

// Property is a property identifier (BACnetPropertyIdentifier).
type Property uint32

// The properties the simulator's objects have, and the three that stand for a
// set in ReadPropertyMultiple.
const (
	PropActiveText            Property = 4
	PropAll                   Property = 8
	PropAPDUTimeout           Property = 11
	PropApplicationVersion    Property = 12
	PropCOVIncrement          Property = 22
	PropDescription           Property = 28
	PropDeviceAddressBinding  Property = 30
	PropEventState            Property = 36
	PropFirmwareRevision      Property = 44
	PropInactiveText          Property = 46
	PropMaxAPDULength         Property = 62
	PropMaxPresValue          Property = 65
	PropMinPresValue          Property = 69
	PropModelName             Property = 70
	PropNumberOfAPDURetries   Property = 73
	PropNumberOfStates        Property = 74
	PropObjectIdentifier      Property = 75
	PropObjectList            Property = 76
	PropObjectName            Property = 77
	PropObjectType            Property = 79
	PropOptional              Property = 80
	PropOutOfService          Property = 81
	PropPolarity              Property = 84
	PropPresentValue          Property = 85
	PropPriorityArray         Property = 87
	PropObjectTypesSupported  Property = 96
	PropServicesSupported     Property = 97
	PropProtocolVersion       Property = 98
	PropReliability           Property = 103
	PropRelinquishDefault     Property = 104
	PropRequired              Property = 105
	PropResolution            Property = 106
	PropSegmentationSupported Property = 107
	PropStateText             Property = 110
	PropStatusFlags           Property = 111
	PropSystemStatus          Property = 112
	PropUnits                 Property = 117
	PropVendorIdentifier      Property = 120
	PropVendorName            Property = 121
	PropProtocolRevision      Property = 139
	PropDatabaseRevision      Property = 155
)

var propertyNames = map[Property]string{
	PropActiveText: "active-text", PropAll: "all", PropAPDUTimeout: "apdu-timeout",
	PropApplicationVersion: "application-software-version", PropCOVIncrement: "cov-increment",
	PropDescription: "description", PropDeviceAddressBinding: "device-address-binding",
	PropEventState: "event-state", PropFirmwareRevision: "firmware-revision", PropInactiveText: "inactive-text",
	PropMaxAPDULength: "max-apdu-length-accepted", PropMaxPresValue: "max-pres-value",
	PropMinPresValue: "min-pres-value", PropModelName: "model-name",
	PropNumberOfAPDURetries: "number-of-apdu-retries", PropNumberOfStates: "number-of-states",
	PropObjectIdentifier: "object-identifier", PropObjectList: "object-list", PropObjectName: "object-name",
	PropObjectType: "object-type", PropOptional: "optional", PropOutOfService: "out-of-service",
	PropPolarity: "polarity", PropPresentValue: "present-value", PropPriorityArray: "priority-array",
	PropObjectTypesSupported: "protocol-object-types-supported", PropServicesSupported: "protocol-services-supported",
	PropProtocolVersion: "protocol-version", PropReliability: "reliability",
	PropRelinquishDefault: "relinquish-default", PropRequired: "required", PropResolution: "resolution",
	PropSegmentationSupported: "segmentation-supported", PropStateText: "state-text",
	PropStatusFlags: "status-flags", PropSystemStatus: "system-status", PropUnits: "units",
	PropVendorIdentifier: "vendor-identifier", PropVendorName: "vendor-name",
	PropProtocolRevision: "protocol-revision", PropDatabaseRevision: "database-revision",
}

// String is the property's name, or property-N for one the simulator has not.
func (p Property) String() string { return name(propertyNames, p, "property") }

// ParseProperty finds a property by name.
func ParseProperty(text string) (Property, bool) { return find(propertyNames, text) }

// PropertyNames lists the names ParseProperty knows, in identifier order.
func PropertyNames() []string { return sortedNames(propertyNames) }

// Values of the enumerations the simulator's properties hold.
const (
	SegmentationNone   = 3 // BACnetSegmentation no-segmentation
	StatusOperational  = 0 // BACnetDeviceStatus operational
	EventStateNormal   = 0
	ReliabilityNoFault = 0
	PolarityNormal     = 0
)
