package plan

// The texts of a batch the station refused with nothing applied.
const (
	// refusedReply is the tool error: the batch id, then the station's decoded text.
	refusedReply = "the station refused batch %s and nothing was applied (a read-back of its targets shows " +
		"no change): it is journaled as refused, with nothing to roll back; fix the request before " +
		"retrying. Station: %s"

	// statedReply is the tool error when the station's own text says it refused before applying anything.
	statedReply = "the station refused batch %s before applying anything and nothing was applied (its own " +
		"error text says so): it is journaled as refused, with nothing to roll back; fix the request before " +
		"retrying. Station: %s"
)
