package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type LocalPrinter struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
}

type LocalPrintJob struct {
	ID            int    `json:"id"`
	DocumentName  string `json:"document_name"`
	UserName      string `json:"user_name"`
	SubmittedTime string `json:"submitted_time"`
	JobStatus     string `json:"job_status"`
	TotalPages    int    `json:"total_pages"`
	Size          int64  `json:"size"`
}

func LocalInstalledPrinters() ([]LocalPrinter, string, error) {
	hostname, _ := os.Hostname()
	if runtime.GOOS != "windows" {
		return nil, hostname, errors.New("mahalliy printerlar faqat Windows serverda ko'rinadi")
	}

	printers, err := listWindowsPrinters()
	if err != nil {
		return nil, hostname, err
	}
	return printers, hostname, nil
}

func LocalPrinterJobs(printerName string) ([]LocalPrintJob, error) {
	if runtime.GOOS != "windows" {
		return nil, errors.New("print queue faqat Windows serverda ko'rinadi")
	}
	if strings.TrimSpace(printerName) == "" {
		return nil, errors.New("printer nomi bo'sh")
	}

	script := `
$printerName = $env:AC_PRINTER_NAME
if ([string]::IsNullOrWhiteSpace($printerName)) {
	throw "printer nomi bo'sh"
}
$jobs = @(Get-PrintJob -PrinterName $printerName -ErrorAction Stop)
if ($jobs.Count -eq 0) {
	return
}
$jobs | ForEach-Object {
	[PSCustomObject]@{
		id = $_.ID
		document_name = $_.DocumentName
		user_name = $_.UserName
		submitted_time = $_.SubmittedTime.ToString("yyyy-MM-dd HH:mm:ss")
		job_status = $_.JobStatus.ToString()
		total_pages = $_.TotalPages
		size = $_.Size
	}
} | ConvertTo-Json -Compress
`

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Env = append(os.Environ(), "AC_PRINTER_NAME="+printerName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if strings.Contains(strings.ToLower(msg), "no msft_printjob objects found") {
			return []LocalPrintJob{}, nil
		}
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("print queue olishda xatolik: %s", msg)
	}

	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return []LocalPrintJob{}, nil
	}

	var jobs []LocalPrintJob
	if strings.HasPrefix(raw, "[") {
		if err := json.Unmarshal([]byte(raw), &jobs); err != nil {
			return nil, err
		}
	} else {
		var single LocalPrintJob
		if err := json.Unmarshal([]byte(raw), &single); err != nil {
			return nil, err
		}
		jobs = []LocalPrintJob{single}
	}

	return jobs, nil
}

func listWindowsPrinters() ([]LocalPrinter, error) {
	script := `
$defaultPrinter = (Get-Printer | Where-Object { $_.Default -eq $true } | Select-Object -First 1 -ExpandProperty Name)
Get-Printer | ForEach-Object {
	[PSCustomObject]@{
		name = $_.Name
		is_default = ($_.Name -eq $defaultPrinter)
	}
} | ConvertTo-Json -Compress
`
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return listWindowsPrintersWmic()
	}

	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return []LocalPrinter{}, nil
	}

	var printers []LocalPrinter
	if strings.HasPrefix(raw, "[") {
		if err := json.Unmarshal([]byte(raw), &printers); err != nil {
			return nil, err
		}
	} else {
		var single LocalPrinter
		if err := json.Unmarshal([]byte(raw), &single); err != nil {
			return nil, err
		}
		printers = []LocalPrinter{single}
	}

	return printers, nil
}

func listWindowsPrintersWmic() ([]LocalPrinter, error) {
	out, err := exec.Command("wmic", "printer", "get", "name").Output()
	if err != nil {
		return nil, fmt.Errorf("printerlar ro'yxatini olishda xatolik: %w", err)
	}

	lines := strings.Split(string(out), "\n")
	printers := []LocalPrinter{}
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name == "" || strings.EqualFold(name, "Name") {
			continue
		}
		printers = append(printers, LocalPrinter{Name: name})
	}
	return printers, nil
}
