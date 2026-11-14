package prepush

// pushedRange is the commit range a ref line pushes. An existing remote ref is gated from the remote
// sha. A new remote ref is gated from where the branch left main: diffing the lone local sha would
// compare it with the working tree (empty after a clean commit) and pass every new branch unchecked.
// ok is false after the refusal is printed.
func (h hook) pushedRange(localSha, remoteSha string) (rng string, ok bool) {
	if !isNewRemoteRef(remoteSha) {
		return remoteSha + ".." + localSha, true
	}
	for _, mainRef := range newBranchBases {
		if base, found := gitOut(nil, "merge-base", mainRef, localSha); found {
			if base == "" {
				break
			}
			return base + ".." + localSha, true
		}
	}
	refuse(h.s.Err, msgNoBase, localSha)
	return "", false
}
