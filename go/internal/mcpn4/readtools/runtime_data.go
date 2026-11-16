package readtools

import "time"

// n4_station_runtime: what the station is running in one read. Its uptime (spy sysInfo, from
// the bootTicks/currentTicks pair), the module parts it has loaded with their versions (spy
// sysManagers/moduleManager) and, with --niagara-home, the module jars in the modules directory
// of that installation, each loaded part beside its jar, so "a restart is needed" is a fact the
// tool states. Measured on Niagara 4.15: the module manager lists only the parts the station
// loaded (311 of 740 jars on the laboratory station), and the session offers no listing of the
// modules directory (n4_read_file reads station-home files only).

// The pages, caps and file names of the tool.
const (
	runtimeSysInfoPage  = "sysInfo"
	runtimeModulesPage  = "sysManagers/moduleManager"
	runtimePageRows     = 5000 // rows read of either page; the module list then caps at max_rows
	runtimeDefaultRows  = 500
	runtimeModulesDir   = "modules"
	runtimeJarSuffix    = ".jar"
	runtimeModuleXML    = "META-INF/module.xml"
	runtimeMaxXMLBytes  = 64 << 10
	runtimeMaxJars      = 5000
	runtimeBootSlack    = 2 * time.Second // a jar this much younger than the computed boot still counts as older
	runtimeStatusOK     = "ok"
	runtimeUnrecognized = "not_recognized"
	runtimeUnavailable  = "unavailable"
)

// The sysInfo properties the uptime section reads, and the reply key of each text one.
const (
	runtimeBootTicks    = "bootTicks"
	runtimeCurrentTicks = "currentTicks"
)

var runtimeUptimeText = [][2]string{{"bootTime", "boot_time"}, {"currentTime", "current_time"}, {"upTime", "up_time"}}

// runtimeModuleColumns are the module manager headers of a part's name, vendor and version.
var runtimeModuleColumns = [3]string{"Module", "Vendor", "Version"}

// runtimeProfiles are the runtime profile suffixes a part name adds to a jar named after its
// module alone (baja.jar holds the part baja-rt).
var runtimeProfiles = []string{"rt", "ux", "wb", "se", "doc"}

// The reasons of a mismatch between what the station loaded and the jars on disk.
const (
	reasonVersionDiffers   = "version_differs"
	reasonModifiedAfter    = "modified_after_boot"
	reasonLoadedNotOnDisk  = "loaded_not_on_disk"
	reasonNewAfterBoot     = "not_loaded_modified_after_boot"
	runtimeUptimeNoTicks   = "the sysInfo page holds no bootTicks/currentTicks pair (the layout measured on Niagara 4.15); read it with n4_spy_page path sysInfo"
	runtimeModulesNoTable  = "the module manager page holds no table with Module, Vendor and Version columns (the layout measured on Niagara 4.15); read it with n4_spy_page path " + runtimeModulesPage
	runtimeNoHome          = "the session lists no module jars (the module manager page lists loaded modules only, n4_read_file reads station-home files only): start the server with --niagara-home %s=PATH, the station's Niagara installation directory reachable from this machine, to compare them"
	runtimeNoDir           = "the modules directory of the --niagara-home of %s cannot be listed: %s"
	runtimeRestartYes      = "%d difference(s) between the loaded modules and the jars on disk: the station loads module jars at boot, so a restart is needed to run what is on disk"
	runtimeRestartNo       = "every loaded module has its jar on disk with the loaded version, and no jar changed after the boot"
	runtimeRestartUnknown  = "not known: %s"
	runtimeUnknownDisk     = "the jars on disk are unknown"
	runtimeUnknownUptime   = "the uptime is unknown, so a jar replaced after the boot cannot be told"
	runtimeUnknownLoaded   = "the loaded module list is unknown or was cut"
	runtimeUnknownJars     = "%d jar(s) could not be read"
	runtimeUnknownScanCut  = "the scan stopped at a cap (%d jars or timeout_s)"
	runtimeLoadedPageCut   = "the module manager page was cut (%s): count is a lower bound"
	runtimeJarTimeLayout   = time.RFC3339
	runtimeUnknownSeparate = "; "
)
