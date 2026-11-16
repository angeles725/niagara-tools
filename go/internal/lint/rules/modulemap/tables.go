package modulemap

// header is the table header the map must carry, cell by cell.
var header = []string{"Concern", "Part", "Files", "Tests", "Change here when"}

// roleFolders are the fixed role folders of each part suffix (types/structure.md § Module map); the concern
// folders under them carry any per-module name (modularity-structure Δ2).
var roleFolders = map[string][]string{
	"-rt": {"model", "adapter"},
	"-ux": {"route", "service", "model"},
	"-wb": {"model"},
	"-se": {"model"},
}
