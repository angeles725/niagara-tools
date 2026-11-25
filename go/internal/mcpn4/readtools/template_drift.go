package readtools

// Template drift of n4_list_templates: whether the deployed copy of a template still has the
// template's signature. Both signatures are TemplateInfo properties the station computes, not
// values it watches. BTemplateService.doUpdateSignatures (BTemplateService.java:284-288) sets
// them for every TemplateInfo by updateSignature (:293-312): localSignature is recomputed from
// the deployed root (UpgradeUtil.getTemplateSignature), ntplSignature is read from the deployed
// .ntpl manifest only while it is at most one character long. The action updateSignatures
// (flags 16, :60, :362) runs when the service is subscribed (:318-324) and in
// BUpgradeTemplateJob (:89), so between runs localSignature is a snapshot and can hide an edit
// of the deployed copy. BTemplateInfo.getModifiedState (BTemplateInfo.java:207-212) calls the
// template's signature unknown when it is at most one character; an empty localSignature is
// no better (the station never computed it), so it is unknown here as well, where the station's
// own comparison would call it a difference.

const (
	ltNoTemplateSigNote = "ntplSignature is empty (one character or less): the station does not know the template's signature, so drift is unknown"
	ltNoLocalSigNote    = "localSignature is empty: the station has not computed the deployed copy's signature (updateSignatures has not run for this instance), so drift is unknown"

	// ltSignatureNote is said once per reply that holds TemplateInfo rows.
	ltSignatureNote = "drift compares ntplSignature with localSignature as the TemplateService last stored them: the station " +
		"refreshes localSignature only when its updateSignatures action runs (when the service is subscribed and in the " +
		"upgrade job), so a stale localSignature can hide an edit of the deployed copy made since. That action is a write " +
		"and is not invoked by this tool"
)

// ltDrift is true when the deployed copy's signature differs from the template's, false when
// they are equal, and nil (with the reason) when either is unknown.
func ltDrift(ntpl, local string) (any, string) {
	switch {
	case len(ntpl) <= 1:
		return nil, ltNoTemplateSigNote
	case local == "":
		return nil, ltNoLocalSigNote
	}
	return ntpl != local, ""
}
