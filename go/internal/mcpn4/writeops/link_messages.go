package writeops

// The dry-run refusals of n4_create_link that come before a token (bench retro 2026-11-15
// delta 4).
const (
	// msgSlotMissing is a link end whose slot neither the component nor its type contract has:
	// the end ("source" or "target"), the slot, the component's ORD and its type.
	msgSlotMissing = "%s slot %s does not exist on %s (type %s): neither the component nor its type " +
		"declares it; check the slot name"
	// msgIdenticalLink is a link already on the target with the same source and slots: the
	// target ORD, the link's name, the source slot and the target slot.
	msgIdenticalLink = "an identical link already exists on %s: %s links the same source slot %s to " +
		"target slot %s; nothing to create"
	// msgLinkCommands is a link from an {ok} source into a writable's control input: the target
	// slot's ORD, the source's value as the station encodes it and its status.
	msgLinkCommands = "this link will command %s to %s %s as soon as it is made"
)
