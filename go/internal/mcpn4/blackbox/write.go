package blackbox

// The write protocol every write tool follows: a dry run returns a plan and a confirmation
// token; executing repeats the arguments with dry_run false and that token.

// Dry is the dry run of a write tool (dry_run is the default); it returns the plan reply.
func (s *Server) Dry(tool string, args Args) map[string]any {
	s.t.Helper()
	return s.OK(tool, args)
}

// Execute is args with dry_run false and token, for a write tool's executing call.
func Execute(args Args, token any) Args {
	out := Args{"dry_run": false, "confirmation_token": token}
	for k, v := range args {
		out[k] = v
	}
	return out
}

// RunWrite is a dry run then its execution with the plan's token; it returns the execution
// reply and fails the test when either call fails.
func (s *Server) RunWrite(tool string, args Args) map[string]any {
	s.t.Helper()
	plan := s.Dry(tool, args)
	return s.OK(tool, Execute(args, plan["confirmation_token"]))
}
