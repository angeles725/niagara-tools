package bacnet

import (
	"cmp"
	"fmt"
	"slices"
)

// Reject is the reason of a Reject PDU (BACnetRejectReason). The decoder
// returns it as the error of a request that is not well formed.
type Reject byte

// The reject reasons the simulator gives.
const (
	RejectOther                Reject = 0
	RejectInvalidParameterType Reject = 3
	RejectInvalidTag           Reject = 4
	RejectMissingParameter     Reject = 5
	RejectParameterOutOfRange  Reject = 6
	RejectTooManyArguments     Reject = 7
	RejectUnrecognizedService  Reject = 9
)

var rejectNames = map[Reject]string{
	RejectOther: "other", RejectInvalidParameterType: "invalid-parameter-data-type", RejectInvalidTag: "invalid-tag",
	RejectMissingParameter: "missing-required-parameter", RejectParameterOutOfRange: "parameter-out-of-range",
	RejectTooManyArguments: "too-many-arguments", RejectUnrecognizedService: "unrecognized-service",
}

func (r Reject) String() string { return name(rejectNames, r, "reject-reason") }
func (r Reject) Error() string  { return "reject: " + r.String() }

// Abort is the reason of an Abort PDU (BACnetAbortReason).
type Abort byte

// The abort reasons the simulator gives.
const (
	AbortOther                    Abort = 0
	AbortBufferOverflow           Abort = 1
	AbortSegmentationNotSupported Abort = 4
)

var abortNames = map[Abort]string{
	AbortOther: "other", AbortBufferOverflow: "buffer-overflow", AbortSegmentationNotSupported: "segmentation-not-supported",
}

func (a Abort) String() string { return name(abortNames, a, "abort-reason") }

// Error is the class and code of an Error PDU, or of one property inside a
// ReadPropertyMultiple acknowledgement.
type Error struct {
	Class uint32
	Code  uint32
}

// The error classes (BACnetErrorClass).
const (
	ClassDevice    = 0
	ClassObject    = 1
	ClassProperty  = 2
	ClassResources = 3
	ClassServices  = 5
)

// The error codes (BACnetErrorCode).
const (
	CodeOperationalProblem      = 25
	CodeUnknownObject           = 31
	CodeUnknownProperty         = 32
	CodeValueOutOfRange         = 37
	CodeWriteAccessDenied       = 40
	CodeInvalidArrayIndex       = 42
	CodeOptionalNotSupported    = 45
	CodeInvalidDataType         = 9
	CodePropertyIsNotAnArray    = 50
	CodeNoSpaceToAddListElement = 19
)

// The errors the simulator answers with.
var (
	ErrOperationalProblem = &Error{ClassDevice, CodeOperationalProblem}
	ErrUnknownObject      = &Error{ClassObject, CodeUnknownObject}
	ErrUnknownProperty    = &Error{ClassProperty, CodeUnknownProperty}
	ErrWriteAccessDenied  = &Error{ClassProperty, CodeWriteAccessDenied}
	ErrValueOutOfRange    = &Error{ClassProperty, CodeValueOutOfRange}
	ErrInvalidArrayIndex  = &Error{ClassProperty, CodeInvalidArrayIndex}
	ErrInvalidDataType    = &Error{ClassProperty, CodeInvalidDataType}
	ErrNotAnArray         = &Error{ClassProperty, CodePropertyIsNotAnArray}
	ErrCOVNotSupported    = &Error{ClassServices, CodeOptionalNotSupported}
	ErrNoSubscriptionRoom = &Error{ClassResources, CodeNoSpaceToAddListElement}
)

var classNames = map[uint32]string{
	ClassDevice: "device", ClassObject: "object", ClassProperty: "property", ClassResources: "resources", 4: "security",
	ClassServices: "services", 6: "vt", 7: "communication",
}

var codeNames = map[uint32]string{
	CodeInvalidDataType: "invalid-data-type", CodeNoSpaceToAddListElement: "no-space-to-add-list-element",
	CodeOperationalProblem: "operational-problem", CodeUnknownObject: "unknown-object",
	CodeUnknownProperty: "unknown-property", CodeValueOutOfRange: "value-out-of-range",
	CodeWriteAccessDenied: "write-access-denied", CodeInvalidArrayIndex: "invalid-array-index",
	CodeOptionalNotSupported: "optional-functionality-not-supported", CodePropertyIsNotAnArray: "property-is-not-an-array",
}

// ClassName names the error class.
func (e Error) ClassName() string { return name(classNames, e.Class, "error-class") }

// CodeName names the error code.
func (e Error) CodeName() string { return name(codeNames, e.Code, "error-code") }

func (e *Error) Error() string { return "error: " + e.ClassName() + " / " + e.CodeName() }

// name is the name of a code, or kind-N for one the tables do not hold.
func name[K cmp.Ordered](names map[K]string, key K, kind string) string {
	if n, ok := names[key]; ok {
		return n
	}
	return fmt.Sprintf("%s-%v", kind, key)
}

// sortedNames lists the names of a table in key order.
func sortedNames[K cmp.Ordered](names map[K]string) []string {
	keys := make([]K, 0, len(names))
	for k := range names {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, names[k])
	}
	return out
}

// find is the key of a name in a table.
func find[K comparable](names map[K]string, wanted string) (K, bool) {
	for k, n := range names {
		if n == wanted {
			return k, true
		}
	}
	var zero K
	return zero, false
}
