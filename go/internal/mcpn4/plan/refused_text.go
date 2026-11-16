package plan

// The texts of a batch the station refused with nothing applied.
const (
	// refusedReply is the tool error: the batch id, then the station's decoded text.
	refusedReply = "the station refused batch %s and nothing was applied (a read-back of its targets shows " +
		"no change): it is journaled as refused, with nothing to roll back; fix the request before " +
		"retrying. Station: %s"
)
