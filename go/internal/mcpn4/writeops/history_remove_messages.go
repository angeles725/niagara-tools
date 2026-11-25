package writeops

// Messages of n4_remove_history, as data.
const (
	msgHRRefused = "n4_remove_history refused, nothing was planned: %s"
	msgHRAck     = "removing a history deletes it and every record in it for good: pass acknowledge_data_loss: true to plan it"
	msgHRNoCall  = "the station client cannot read histories or call the history channel"
	msgHRNoExt   = "no history extension records into %s: its slot path is unknown, so it is outside every write scope " +
		"I can prove (an imported or archived history); remove it in Workbench"
	msgHRCut = "the history extension read was cut after %d rows, so the extensions recording into %s cannot be " +
		"proven; nothing was planned"
	msgHRSlot    = "extension slot %s is not a plain slot path, so its write scope cannot be checked"
	msgHROutside = "extension %s is outside the write scope: remove the history only from a path the scope covers"
	msgHREnabled = "extension %s is enabled and still records into it; pass allow_enabled_extension: true to remove it " +
		"anyway (the extension keeps running and starts a new, empty history at its next record)"
	noteHRNoInverse = "no inverse: the history and all its records are deleted on the station; the records are gone " +
		"and n4_rollback cannot bring them back"
	noteHRExt        = "recorded by extension %s (enabled: %t); the extension itself is not removed"
	noteHREnabledExt = "ENABLED extension %s keeps recording: it creates a new, empty history at its next record"
	msgHRCount       = "the record count did not answer: %s"
	msgHRCountShape  = "the record count answered no whole number"
)
