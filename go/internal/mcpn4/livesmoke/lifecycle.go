package livesmoke

import "github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"

// Save saves the station; folderPresent records whether the saved station holds the scratch
// folder, so an abort after it saves again once the folder is removed.
func (s *Scenario) Save(folderPresent bool) func(*pyjson.Object) (any, any, error) {
	return func(entry *pyjson.Object) (any, any, error) {
		out, err := s.Write(entry, "n4_save_station", nil, pyjson.NewObject())
		if err != nil {
			return nil, nil, err
		}
		persisted, err := get(out, "persisted")
		if err != nil {
			return nil, nil, err
		}
		verdict := "unverified"
		switch persisted {
		case true:
			verdict = "verified"
			s.savedWithFolder = folderPresent
		case false:
			verdict = "mismatch"
		}
		evidence, _ := get(out, "evidence")
		return verdict, kwargs("persisted", persisted, "evidence", evidence), nil
	}
}

// SavedWithFolder reports whether the last verified save held the scratch folder.
func (s *Scenario) SavedWithFolder() bool { return s.savedWithFolder }

func (s *Scenario) remove(remember bool) stepFn {
	return func(entry *pyjson.Object) (any, any, error) {
		out, err := s.Write(entry, "n4_remove_component", nil, kwargs("parent_ord", RootOrd, "name", Scratch))
		if err != nil {
			return nil, nil, err
		}
		if remember {
			s.removalBatch, _ = entry.Get("batch_id")
		}
		return verdictOf(out)
	}
}

func (s *Scenario) rollback(entry *pyjson.Object) (any, any, error) {
	if !pyjson.Truthy(s.removalBatch) {
		return "skipped", "no removal batch was recorded", nil
	}
	out, err := s.Write(entry, "n4_rollback", nil, kwargs("batch_id", s.removalBatch))
	if err != nil {
		return nil, nil, err
	}
	relinks, err := get(out, "relinks")
	if err != nil {
		return nil, nil, err
	}
	return verdictOf(out, "relinks", relinks)
}

func (s *Scenario) removeAgain(entry *pyjson.Object) (any, any, error) {
	present, err := s.scratchPresent()
	if err != nil {
		return nil, nil, err
	}
	if !present {
		return "skipped", "folder absent after the rollback", nil
	}
	return s.remove(false)(entry)
}

func (s *Scenario) folderAbsent(*pyjson.Object) (any, any, error) {
	present, err := s.scratchPresent()
	if err != nil {
		return nil, nil, err
	}
	verdict := "verified"
	if present {
		verdict = "failed"
	}
	return verdict, kwargs("present", present), nil
}

// cleanup is the best effort after an abort: remove the folder the station may hold, and save
// again when a saved station held it.
func (s *Scenario) cleanup() {
	s.aborted = false
	present, known := true, false // unknown (the probe failed) counts as present
	s.step("cleanup: folder present?", "n4_navigate", func(*pyjson.Object) (any, any, error) {
		p, err := s.scratchPresent()
		if err != nil {
			return nil, nil, err
		}
		present, known = p, true
		return "verified", kwargs("present", p), nil
	}, false, false)
	if present || !known {
		s.step("cleanup", "n4_remove_component", s.remove(false), false, false)
	}
	if s.savedWithFolder {
		s.step("cleanup save", "n4_save_station", s.Save(false), false, false)
	}
}
