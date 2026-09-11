//go:build windows

package utils

import (
	"fmt"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procEnumJobsW = modWinspool.NewProc("EnumJobsW")
	procSetJobW   = modWinspool.NewProc("SetJobW")
)

// JOB_INFO_1W status bits (winspool.h).
const (
	jobStatusPaused           = 0x00000001
	jobStatusError            = 0x00000002
	jobStatusDeleting         = 0x00000004
	jobStatusSpooling         = 0x00000008
	jobStatusPrinting         = 0x00000010
	jobStatusOffline          = 0x00000020
	jobStatusPaperOut         = 0x00000040
	jobStatusPrinted          = 0x00000080
	jobStatusDeleted          = 0x00000100
	jobStatusBlockedDevQ      = 0x00000200
	jobStatusUserIntervention = 0x00000400
	jobStatusRestart          = 0x00000800
)

const (
	jobControlDelete = 5
	jobInfoLevel1    = 1
)

type systemTime struct {
	Year         uint16
	Month        uint16
	DayOfWeek    uint16
	Day          uint16
	Hour         uint16
	Minute       uint16
	Second       uint16
	Milliseconds uint16
}

type jobInfo1W struct {
	JobId        uint32
	pPrinterName *uint16
	pMachineName *uint16
	pUserName    *uint16
	pDocument    *uint16
	pDatatype    *uint16
	pStatus      *uint16
	Status       uint32
	Priority     uint32
	Position     uint32
	TotalPages   uint32
	PagesPrinted uint32
	Submitted    systemTime
}

func utf16PtrToString(p *uint16) string {
	if p == nil {
		return ""
	}
	return windows.UTF16PtrToString(p)
}

func jobStatusString(status uint32, pStatus *uint16) string {
	if s := strings.TrimSpace(utf16PtrToString(pStatus)); s != "" {
		return s
	}
	parts := make([]string, 0, 4)
	add := func(bit uint32, name string) {
		if status&bit != 0 {
			parts = append(parts, name)
		}
	}
	add(jobStatusError, "Error")
	add(jobStatusPaused, "Paused")
	add(jobStatusOffline, "Offline")
	add(jobStatusPaperOut, "PaperOut")
	add(jobStatusUserIntervention, "UserIntervention")
	add(jobStatusBlockedDevQ, "Blocked")
	add(jobStatusSpooling, "Spooling")
	add(jobStatusPrinting, "Printing")
	add(jobStatusRestart, "Restarting")
	add(jobStatusPrinted, "Printed")
	add(jobStatusDeleting, "Deleting")
	add(jobStatusDeleted, "Deleted")
	if len(parts) == 0 {
		return "Unknown"
	}
	return strings.Join(parts, ",")
}

func submittedTimeString(st systemTime) string {
	if st.Year == 0 {
		return ""
	}
	t := time.Date(int(st.Year), time.Month(st.Month), int(st.Day), int(st.Hour), int(st.Minute), int(st.Second), int(st.Milliseconds)*1e6, time.Local)
	return t.Format("2006-01-02 15:04:05")
}

func localPrinterJobsWin32(printerName string) ([]LocalPrintJob, error) {
	printerNamePtr, err := windows.UTF16PtrFromString(printerName)
	if err != nil {
		return nil, err
	}
	var printerHandle windows.Handle
	ret, _, callErr := procOpenPrinterW.Call(
		uintptr(unsafe.Pointer(printerNamePtr)),
		uintptr(unsafe.Pointer(&printerHandle)),
		0,
	)
	if ret == 0 {
		return nil, fmt.Errorf("OpenPrinter: %w", callErr)
	}
	defer procClosePrinter.Call(uintptr(printerHandle))

	var needed, returned uint32
	ret, _, callErr = procEnumJobsW.Call(
		uintptr(printerHandle),
		0,
		0xffffffff,
		jobInfoLevel1,
		0,
		0,
		uintptr(unsafe.Pointer(&needed)),
		uintptr(unsafe.Pointer(&returned)),
	)
	if needed == 0 {
		return []LocalPrintJob{}, nil
	}
	buf := make([]byte, needed)
	ret, _, callErr = procEnumJobsW.Call(
		uintptr(printerHandle),
		0,
		0xffffffff,
		jobInfoLevel1,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(needed),
		uintptr(unsafe.Pointer(&needed)),
		uintptr(unsafe.Pointer(&returned)),
	)
	if ret == 0 {
		return nil, fmt.Errorf("EnumJobs: %w", callErr)
	}
	if returned == 0 {
		return []LocalPrintJob{}, nil
	}

	jobs := make([]LocalPrintJob, 0, returned)
	stride := unsafe.Sizeof(jobInfo1W{})
	for i := uint32(0); i < returned; i++ {
		info := (*jobInfo1W)(unsafe.Pointer(&buf[uintptr(i)*stride]))
		jobs = append(jobs, LocalPrintJob{
			ID:            int(info.JobId),
			DocumentName:  utf16PtrToString(info.pDocument),
			UserName:      utf16PtrToString(info.pUserName),
			SubmittedTime: submittedTimeString(info.Submitted),
			JobStatus:     jobStatusString(info.Status, info.pStatus),
			TotalPages:    int(info.TotalPages),
			PagesPrinted:  int(info.PagesPrinted),
			// JOB_INFO_1 has no byte Size; keep 0 (PowerShell fallback may fill Size).
			Size: 0,
		})
	}
	return jobs, nil
}

func cancelPrintJobByIDWin32(printerName string, jobID int) error {
	if jobID <= 0 {
		return nil
	}
	printerNamePtr, err := windows.UTF16PtrFromString(printerName)
	if err != nil {
		return err
	}
	var printerHandle windows.Handle
	ret, _, callErr := procOpenPrinterW.Call(
		uintptr(unsafe.Pointer(printerNamePtr)),
		uintptr(unsafe.Pointer(&printerHandle)),
		0,
	)
	if ret == 0 {
		return fmt.Errorf("OpenPrinter: %w", callErr)
	}
	defer procClosePrinter.Call(uintptr(printerHandle))

	ret, _, callErr = procSetJobW.Call(
		uintptr(printerHandle),
		uintptr(uint32(jobID)),
		0,
		0,
		uintptr(jobControlDelete),
	)
	if ret == 0 {
		return fmt.Errorf("SetJob(DELETE) #%d: %w", jobID, callErr)
	}
	return nil
}
