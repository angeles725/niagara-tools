package readtoolstest

import "sync"

// ordMu guards Station.OrdRequests: the histories audit reads several ORDs at once (readtools
// runPool), so a fake station is called from more than one goroutine.
var ordMu sync.Mutex

// noteOrd records one GET /ord/ that reached the station.
func (s *Station) noteOrd(r OrdRequest) {
	ordMu.Lock()
	defer ordMu.Unlock()
	s.OrdRequests = append(s.OrdRequests, r)
}
