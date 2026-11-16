package rcscan

const (
	fetchMsg = "fetch( with no signal/timeout -- route it through the fetchT helper (AbortController + timeout)"
	gateMsg  = "control disabled in a login-gated file -- a disabled control fires no click, so the gate never opens; mark it (class / aria-disabled) and route the click to the gate"
	pollMsg  = " -- self-schedule with setTimeout after the poll settles (+ inFlight guard)"
)

// dataBudget is the largest data: URI (in chars) an rc file may inline.
const dataBudget = 20480

// maxJSLines is the js-file-size budget (modularity-gaps Δ3): a .js file over it is split by concern.
const maxJSLines = 400
