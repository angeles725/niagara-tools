package reqlog

// ProtocolBACnetIP is the Protocol of a BACnet entry. A Modbus entry has no
// protocol field: its lines are as they were before BACnet was added.
const ProtocolBACnetIP = "bacnet-ip"

// The values of BACnet.Result that a Modbus entry does not have.
const (
	ResultError   = "error"   // an Error PDU: ErrorClass and ErrorCode
	ResultReject  = "reject"  // a Reject PDU: Reason
	ResultAbort   = "abort"   // an Abort PDU: Reason
	ResultIgnored = "ignored" // a frame a device does not answer: Reason
	ResultRefused = "refused" // a BVLC-Result that is not success: Reason
	ResultSent    = "sent"    // a frame the simulator originated (a COV notification)
)

// BACnet is one BACnet/IP frame the simulator received and its outcome, or one
// it originated.
type BACnet struct {
	Time       string   `json:"time"`       // when it was handled, RFC 3339 UTC
	ElapsedMS  int64    `json:"elapsed_ms"` // since the simulator started
	Protocol   string   `json:"protocol"`
	Remote     string   `json:"remote"` // the client's address
	Device     string   `json:"device"` // the scenario name of the device
	Link       string   `json:"link"`   // the BVLL function
	PDU        string   `json:"pdu,omitempty"`
	Service    string   `json:"service,omitempty"`
	InvokeID   *int     `json:"invoke_id,omitempty"`
	Object     string   `json:"object,omitempty"` // "type,instance"
	Property   string   `json:"property,omitempty"`
	Index      *int     `json:"index,omitempty"`    // the array index of the property
	Priority   *int     `json:"priority,omitempty"` // of a write
	Written    string   `json:"written,omitempty"`  // the value of a write; "null" relinquishes
	Detail     string   `json:"detail,omitempty"`   // what else was asked, as text
	Result     string   `json:"result"`
	ErrorClass string   `json:"error_class,omitempty"`
	ErrorCode  string   `json:"error_code,omitempty"`
	Reason     string   `json:"reason,omitempty"`
	Read       []string `json:"read,omitempty"` // the values a ReadProperty returned
	To         []string `json:"to,omitempty"`   // where an I-Am, an I-Have or a notification was sent
	Duplicate  bool     `json:"duplicate,omitempty"`
	DelayMS    int64    `json:"delay_ms,omitempty"` // how long the response was held
}

// LogBACnet writes one BACnet entry; like Log, it never fails.
func (w *Writer) LogBACnet(e BACnet) {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	_ = w.enc.Encode(e)
}
