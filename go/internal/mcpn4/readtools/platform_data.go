package readtools

// What n4_station_health platform=true reads and says (data only). Every page is a spy page,
// read like n4_spy_page reads it (a station user allowed to read spy pages); the CPU, memory
// and licence items reuse the system and license sections, with no request of their own.

// The pages, by their path below the spy root, as the 4.15 sources register them.
const (
	platformSysInfoPage = "sysInfo"                        // SysInfoSpy: Performance totalMemory/freeMemory/usedMemory
	platformJmxPage     = "util/JMX Info"                  // UtilSpy.JmxInfo: one table per memory pool
	platformHistoryPage = "localHistory"                   // BLocalHistoryDatabase.SpyLocalHistoryDatabase
	platformCryptoPage  = "securityInfo/Cryptography Info" // CryptoPlatformPage: the trust stores
	platformPageRows    = 20000                            // rows kept per page: more than a byte-capped page holds
)

// The defaults and caps of the arguments and lists.
const (
	platformWarnDays     = 30   // expiry_warn_days
	platformMaxCerts     = 50   // certificates listed, nearest expiry first
	platformMaxModules   = 1000 // modules listed with include_modules
	platformHeapWarnFrac = 0.9  // a heap this full is warned about
)

// The columns, titles and properties the pages are read by.
const (
	sysInfoTotal         = "totalMemory"
	sysInfoFree          = "freeMemory"
	jmxTitleSuffix       = " (KB)"
	jmxMaxColumn         = "max"
	historyNameColumn    = "Name"
	historyRecordsColumn = "Records"
	certAliasColumn      = "Alias"
	certIssuerColumn     = "Issued By"
	certExpiresColumn    = "Expires"
)

// heapPoolNames are the parts of a JVM memory pool name that make it a heap pool (the young,
// survivor and old generations of the HotSpot collectors, and the single-pool collectors);
// every other pool (Metaspace, Code Cache, CodeHeap, Compressed Class Space) is not heap.
var heapPoolNames = []string{"Eden Space", "Survivor Space", "Old Gen", "Tenured Gen", "ZHeap", "Shenandoah"}

// certStores maps the title of a Cryptography Info table to the store its rows are in.
var certStores = [][2]string{{"Trusted System Certificates", "system"}, {"Trusted User Certificates", "user"}}

// The sources and notes of the platform section.
const (
	heapSource        = "spy:/" + platformSysInfoPage + ", spy:/" + platformJmxPage
	historySource     = "spy:/" + platformHistoryPage
	certSource        = "spy:/" + platformCryptoPage
	modulesSource     = "spy:/" + runtimeModulesPage
	platformPageNote  = "the spy page %s did not answer (status %v, HTTP %v)%s"
	platformPageCut   = "the spy page %s was cut (%s): %s"
	heapNoMemory      = "the sysInfo page holds no totalMemory/freeMemory pair in KB (SysInfoSpy, 4.15)"
	heapNoMax         = "the JMX Info page has no heap pool with a defined maximum (pools seen: %d)"
	historyNoTable    = "the localHistory page holds no table with Name and Records columns (BLocalHistoryDatabase, 4.15)"
	historyBytesNote  = "no read-only page or slot gives the size in bytes of the history files: the localHistory spy page gives record counts only; route: the size of the station's history directory (dbDir) on the host, through the platform's file transfer or a shell"
	historyUnreadNote = "%d history row(s) whose Records cell is not a number are counted in histories but not in records"
	certNoTable       = "the Cryptography Info page holds no Trusted System or Trusted User Certificates table with Alias and Expires columns (CryptoPlatformPage, 4.15)"
	certScopeNote     = "certificates are the trust stores (system and user) of the Cryptography Info spy page; the station's own key store certificates (the TLS server certificate of WebService and FoxService) are on no read-only page: read them in Workbench Certificate Management (platform) or the Security Dashboard view"
	certUnreadNote    = "%d certificate(s) whose Expires cell is not a date are listed with expires null"
	certCutNote       = "%d certificate(s) listed of %d (nearest expiry first)"
	cpuNote           = "the system section did not answer: %v"
	licenceNote       = "the license section did not answer: %v"
	heapWarning       = "heap %s used (%d of %d KB)"
	licenceWarnSoon   = "licence: %s %s expires %s, in %d day(s) (window %d)"
	licenceWarnPast   = "licence: %s %s expired %s, %d day(s) ago"
	certWarnSoon      = "certificate %s (%s store) expires %s, in %d day(s) (window %d)"
	certWarnPast      = "certificate %s (%s store) expired %s, %d day(s) ago"
	certWarnMore      = "%d more certificate(s) expired or within %d day(s) are in certificates"
	platformWarnCerts = 10 // certificate warnings given one by one
)
