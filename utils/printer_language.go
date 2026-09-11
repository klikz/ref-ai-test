package utils

import (
	"encoding/json"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

const (
	PrintLanguageGDI  = "gdi"
	PrintLanguageTSPL = "tspl"
	PrintLanguageZPL  = "zpl"
)

type windowsPrinterInfo struct {
	DriverName string `json:"driver_name"`
	PortName   string `json:"port_name"`
	Comment    string `json:"comment"`
	Location   string `json:"location"`
	ShareName  string `json:"share_name"`
}

type printerLanguageCacheEntry struct {
	language   string
	driverName string
	reason     string
}

var printerLanguageCache sync.Map // printer name -> printerLanguageCacheEntry

func officePrinterNameFastPath(printerName string) (bool, string) {
	printerLower := strings.ToLower(strings.TrimSpace(printerName))
	markers := []string{
		"canon", "mf4", "mf3", "imageclass", "brother", "hp laserjet", "hp color",
		"xerox", "kyocera", "ricoh", "samsung scx", "samsung m", "epson l", "officejet",
	}
	for _, marker := range markers {
		if strings.Contains(printerLower, marker) {
			return true, "ofis printeri (nom bo'yicha) — GDI"
		}
	}
	return false, ""
}

// Gprinter / Gainscha / XPrinter — native TSPL only for these brands.
// Other printers keep existing Zebra ZPL / TSC TSPL / GDI (Windows settings) behavior.
func matchGprinterOrXPrinter(driverLower, printerLower string) (string, bool) {
	keywords := []string{"gprinter", "gainscha", "xprinter"}
	blob := driverLower + " | " + printerLower
	for _, keyword := range keywords {
		if strings.Contains(blob, keyword) {
			return keyword, true
		}
	}
	return "", false
}

func NormalizePrintLanguage(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case PrintLanguageTSPL:
		return PrintLanguageTSPL
	case PrintLanguageZPL:
		return PrintLanguageZPL
	default:
		return PrintLanguageGDI
	}
}

func PrintLanguageLabel(language string) string {
	switch NormalizePrintLanguage(language) {
	case PrintLanguageTSPL:
		return "TSPL (Win32 RAW)"
	case PrintLanguageZPL:
		return "ZPL (Win32 RAW)"
	default:
		return "GDI (Windows spooler)"
	}
}

// DetectPrinterLanguage resolves print language from Windows printer driver properties.
func DetectPrinterLanguage(printerName string) (language string, driverName string, reason string) {
	printerName = strings.TrimSpace(printerName)
	if cached, ok := printerLanguageCache.Load(printerName); ok {
		entry := cached.(printerLanguageCacheEntry)
		return entry.language, entry.driverName, entry.reason
	}

	language = PrintLanguageGDI
	driverName = ""
	reason = "standart Windows printer (GDI)"

	upperName := strings.ToUpper(printerName)
	if strings.HasPrefix(upperName, "AC-FILE") {
		switch {
		case strings.Contains(upperName, "ZPL"):
			language, reason = PrintLanguageZPL, "AC-FILE preview (ZPL → PNG)"
		case strings.Contains(upperName, "TSPL"):
			language, reason = PrintLanguageTSPL, "AC-FILE preview (TSPL → PNG)"
		default:
			language, reason = PrintLanguageGDI, "AC-FILE preview (PNG)"
		}
		storePrinterLanguageCache(printerName, language, driverName, reason)
		return language, driverName, reason
	}

	if runtime.GOOS != "windows" || printerName == "" {
		return language, driverName, reason
	}

	printerLower := strings.ToLower(printerName)

	// Prefer brand name match so Gprinter/XPrinter are not forced to GDI
	// when Windows reports a generic/IPP class driver.
	if keyword, ok := matchGprinterOrXPrinter("", printerLower); ok {
		language, reason = PrintLanguageTSPL, "Gprinter/XPrinter (nom): "+keyword
		storePrinterLanguageCache(printerName, language, driverName, reason)
		return language, driverName, reason
	}

	if ok, fastReason := officePrinterNameFastPath(printerName); ok {
		language, driverName, reason = PrintLanguageGDI, "", fastReason
		printerLanguageCache.Store(printerName, printerLanguageCacheEntry{
			language: language, driverName: driverName, reason: reason,
		})
		return language, driverName, reason
	}

	info, err := windowsPrinterInfoQuery(printerName)
	if err != nil || info == nil {
		printerLanguageCache.Store(printerName, printerLanguageCacheEntry{
			language: language, driverName: driverName, reason: reason,
		})
		return language, driverName, reason
	}

	driverName = strings.TrimSpace(info.DriverName)
	driverLower := strings.ToLower(driverName)

	if keyword, ok := matchGprinterOrXPrinter(driverLower, printerLower); ok {
		language, reason = PrintLanguageTSPL, "Gprinter/XPrinter: "+keyword
		storePrinterLanguageCache(printerName, language, driverName, reason)
		return language, driverName, reason
	}

	if isOfficePrinter(driverLower, printerLower) {
		language, reason = PrintLanguageGDI, "ofis printeri (masalan Canon/HP) — faqat GDI, etiketka printeri tavsiya etiladi"
		storePrinterLanguageCache(printerName, language, driverName, reason)
		return language, driverName, reason
	}

	if isGenericWindowsDriver(driverLower) {
		language, reason = PrintLanguageGDI, "generic/Microsoft drayver — faqat GDI"
		storePrinterLanguageCache(printerName, language, driverName, reason)
		return language, driverName, reason
	}

	if keyword, ok := matchDriverKeyword(driverLower, []string{
		"zdesigner", "zebra", "ztc", "zt230", "zt410", "zt411", "zt421", "zd420", "gk420", "zq520",
	}); ok {
		if supportsRawPassthrough(driverName, PrintLanguageZPL) {
			language, reason = PrintLanguageZPL, "Zebra drayveri: "+keyword
		} else {
			language, reason = PrintLanguageGDI, "Zebra drayveri aniqlandi, lekin RAW qo'llab-quvvatlanmaydi — GDI"
		}
		storePrinterLanguageCache(printerName, language, driverName, reason)
		return language, driverName, reason
	}

	if keyword, ok := matchDriverKeyword(driverLower, []string{
		"tsc", "ttp-", "te200", "te300", "ta200", "ta300", "tx200", "tx300", "mh series", "godex", "argox",
	}); ok {
		if supportsRawPassthrough(driverName, PrintLanguageTSPL) {
			language, reason = PrintLanguageTSPL, "TSPL drayveri: "+keyword
		} else {
			language, reason = PrintLanguageGDI, "TSPL drayveri aniqlandi, lekin RAW qo'llab-quvvatlanmaydi — GDI"
		}
		storePrinterLanguageCache(printerName, language, driverName, reason)
		return language, driverName, reason
	}

	language, reason = PrintLanguageGDI, "drayver: "+defaultText(driverName, "noma'lum")+" — GDI"
	storePrinterLanguageCache(printerName, language, driverName, reason)
	return language, driverName, reason
}

func storePrinterLanguageCache(printerName, language, driverName, reason string) {
	printerLanguageCache.Store(printerName, printerLanguageCacheEntry{
		language: language, driverName: driverName, reason: reason,
	})
}

// InvalidatePrinterLanguageCache clears cached detect results (all or one printer).
func InvalidatePrinterLanguageCache(printerName string) {
	printerName = strings.TrimSpace(printerName)
	if printerName == "" {
		printerLanguageCache.Range(func(key, _ any) bool {
			printerLanguageCache.Delete(key)
			return true
		})
		return
	}
	printerLanguageCache.Delete(printerName)
}

func ResolvePrinterLanguage(printerName string) string {
	language, _, _ := DetectPrinterLanguage(printerName)
	return language
}

// ProductionPrintLanguageHint returns operator-facing guidance for a language.
func ProductionPrintLanguageHint(language string) string {
	switch NormalizePrintLanguage(language) {
	case PrintLanguageZPL:
		return "ZPL RAW — darkness/speed shablon sozlamalaridan (Windows Preferences emas). Production uchun tavsiya (Zebra)."
	case PrintLanguageTSPL:
		return "TSPL RAW — density/speed/gap shablon sozlamalaridan. Production uchun tavsiya (Gprinter/XPrinter/TSC)."
	default:
		return "GDI — Windows Printing Preferences. Zebra/Gprinter production uchun fallback; Test Print yoki «Windows sozlamalarini yangilash» kerak bo'lishi mumkin."
	}
}

func RawSpoolDatatype(language string) string {
	// Zebra/TSC Windows drivers accept RAW passthrough; "ZPL" is often not registered.
	_ = language
	return "RAW"
}

func supportsRawPassthrough(driverName, language string) bool {
	if isGenericWindowsDriver(strings.ToLower(driverName)) {
		return false
	}

	driver := strings.ToLower(driverName)
	switch NormalizePrintLanguage(language) {
	case PrintLanguageZPL:
		return strings.Contains(driver, "zebra") ||
			strings.Contains(driver, "zdesigner") ||
			strings.Contains(driver, "ztc")
	case PrintLanguageTSPL:
		return strings.Contains(driver, "tsc") ||
			strings.Contains(driver, "godex") ||
			strings.Contains(driver, "argox") ||
			strings.Contains(driver, "gprinter") ||
			strings.Contains(driver, "gainscha") ||
			strings.Contains(driver, "xprinter")
	default:
		return false
	}
}

func isGenericWindowsDriver(driverLower string) bool {
	genericMarkers := []string{
		"microsoft", "generic", "ipp class", "pdf", "onenote", "xps",
		"fax", "send to", "anydesk", "redirected",
	}
	for _, marker := range genericMarkers {
		if strings.Contains(driverLower, marker) {
			return true
		}
	}
	return false
}

func isOfficePrinter(driverLower, printerLower string) bool {
	markers := []string{
		"canon", "mf4", "mf3", "imageclass", "brother", "hp laserjet", "hp color",
		"xerox", "kyocera", "ricoh", "samsung scx", "samsung m", "epson l", "officejet",
	}
	blob := driverLower + " | " + printerLower
	for _, marker := range markers {
		if strings.Contains(blob, marker) {
			return true
		}
	}
	return false
}

func matchDriverKeyword(driverLower string, keywords []string) (string, bool) {
	for _, keyword := range keywords {
		if strings.Contains(driverLower, keyword) {
			return keyword, true
		}
	}
	return "", false
}

func defaultText(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func windowsPrinterInfoQuery(printerName string) (*windowsPrinterInfo, error) {
	script := `
$printerName = $env:AC_PRINTER_NAME
$escaped = $printerName.Replace("'", "''")
$p = Get-CimInstance -ClassName Win32_Printer -Filter ("Name='" + $escaped + "'") -ErrorAction SilentlyContinue
if ($null -eq $p) { return }
[PSCustomObject]@{
	driver_name = $p.DriverName
	port_name = $p.PortName
	comment = $p.Comment
	location = $p.Location
	share_name = $p.ShareName
} | ConvertTo-Json -Compress
`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Env = append(cmd.Environ(), "AC_PRINTER_NAME="+printerName)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}

	info := windowsPrinterInfo{}
	if err := json.Unmarshal([]byte(raw), &info); err != nil {
		return nil, err
	}
	return &info, nil
}
