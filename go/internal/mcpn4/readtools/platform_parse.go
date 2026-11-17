package readtools

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/spy"
)

// The text the platform pages write, read as numbers and dates (pure functions).

// kilobytes reads a sysInfo memory value: a whole number of KB followed by "kb"
// (SysInfoSpy writes totalMem + "kb").
func kilobytes(text string) (int64, bool) {
	digits, ok := strings.CutSuffix(strings.TrimSpace(text), "kb")
	if !ok || digits == "" {
		return 0, false
	}
	n, err := strconv.ParseInt(digits, 10, 64)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

// certificateDay is the calendar day of an Expires cell, as yyyy-mm-dd: the cell is a
// java.util.Date as text ("Sat Mar 31 17:59:59 CST 2027", always English month names), read by
// its month, day and year fields only, so the time zone written never matters.
func certificateDay(text string) (string, bool) {
	f := strings.Fields(text)
	if len(f) < 4 {
		return "", false
	}
	day, err := time.Parse("Jan 2 2006", f[1]+" "+f[2]+" "+f[len(f)-1])
	if err != nil {
		return "", false
	}
	return day.Format("2006-01-02"), true
}

// heapPool is the pool a JMX Info table is about when that pool is a heap pool.
func heapPool(title string) (string, bool) {
	name, ok := strings.CutSuffix(title, jmxTitleSuffix)
	if !ok {
		return "", false
	}
	for _, part := range heapPoolNames {
		if strings.Contains(name, part) {
			return name, true
		}
	}
	return "", false
}

// heapMax is the heap maximum in KB: the sum of the "current" row max of every heap pool whose
// maximum is defined (the JVM's -1, undefined, is printed 0 after the division by 1024), and
// how many heap pools were seen. ok is false when no heap pool has a readable defined maximum
// or one heap pool's maximum is not a number: then the maximum is unknown, never a guess.
func heapMax(tables []spy.Table) (int64, int, bool) {
	var sum int64
	pools, defined := 0, 0
	for _, tb := range tables {
		if _, ok := heapPool(tb.Title); !ok {
			continue
		}
		col := slices.Index(tb.Header, jmxMaxColumn)
		if col < 0 || len(tb.Rows) == 0 || len(tb.Rows[0]) <= col {
			return 0, pools, false
		}
		pools++
		n, err := strconv.ParseInt(strings.TrimSpace(tb.Rows[0][col]), 10, 64)
		if err != nil {
			return 0, pools, false
		}
		if n > 0 {
			sum, defined = sum+n, defined+1
		}
	}
	return sum, pools, defined > 0
}

// heapFraction is used over max to three decimals, nil when max is not known.
func heapFraction(used, max int64) any {
	if max <= 0 {
		return nil
	}
	return math.Round(float64(used)/float64(max)*1000) / 1000
}

// historyCount is what the localHistory page counts: the histories it lists, their records
// summed, and the rows whose Records cell is not a number (listed, not summed).
type historyCount struct {
	histories, unreadable int
	records               int64
}

// historyRecords reads the table of the localHistory page with Name and Records columns; ok is
// false when the page has none.
func historyRecords(tables []spy.Table) (historyCount, bool) {
	for _, tb := range tables {
		name, rec := slices.Index(tb.Header, historyNameColumn), slices.Index(tb.Header, historyRecordsColumn)
		if name < 0 || rec < 0 {
			continue
		}
		var c historyCount
		for _, row := range tb.Rows {
			if len(row) != len(tb.Header) {
				continue
			}
			c.histories++
			if n, err := strconv.ParseInt(strings.TrimSpace(row[rec]), 10, 64); err == nil && n >= 0 {
				c.records += n
			} else {
				c.unreadable++
			}
		}
		return c, true
	}
	return historyCount{}, false
}

// percent is a fraction as the text of a warning ("97.3%").
func percent(fraction float64) string { return fmt.Sprintf("%.1f%%", fraction*100) }

// expiryDays reads the value of a CertificateExpiryPoint as the station writes it: the days
// to the certificate's notAfter (a whole number, written with the locale's decimal and
// grouping marks, "329,0" or "1.234,0") and its status in braces ("329,0 {ok}").
func expiryDays(text string) (int, string, bool) {
	number, status, _ := strings.Cut(strings.TrimSpace(text), " {")
	status = strings.TrimSuffix(status, "}")
	number = strings.TrimSpace(number)
	if i := strings.LastIndexAny(number, ".,"); i >= 0 && len(number)-i-1 < 3 {
		number = number[:i] // the fraction: the point is a whole number of days
	}
	number = strings.NewReplacer(".", "", ",", "", " ", "", " ", "").Replace(number)
	days, err := strconv.Atoi(number)
	if err != nil {
		return 0, "", false
	}
	return days, status, true
}
