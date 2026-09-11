//go:build windows

package utils

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modWinspool           = windows.NewLazySystemDLL("winspool.drv")
	procOpenPrinterW      = modWinspool.NewProc("OpenPrinterW")
	procClosePrinter      = modWinspool.NewProc("ClosePrinter")
	procStartDocPrinterW  = modWinspool.NewProc("StartDocPrinterW")
	procEndDocPrinter     = modWinspool.NewProc("EndDocPrinter")
	procStartPagePrinter  = modWinspool.NewProc("StartPagePrinter")
	procEndPagePrinter    = modWinspool.NewProc("EndPagePrinter")
	procWritePrinter      = modWinspool.NewProc("WritePrinter")
)

type docInfo1 struct {
	DocName    *uint16
	OutputFile *uint16
	Datatype   *uint16
}

// rawPrintWindows submits a RAW job and returns the spooler job id, which the
// caller needs to verify exactly this job instead of matching document names.
func rawPrintWindows(printerName, docName, datatype string, payload []byte) (uint32, error) {
	jobID, err := rawPrintWindowsDatatype(printerName, docName, datatype, payload)
	if err == nil {
		return jobID, nil
	}
	// Some drivers (e.g. Zebra ZT411) reject "ZPL" but accept RAW passthrough.
	if datatype != "RAW" && isInvalidSpoolDatatypeError(err) {
		return rawPrintWindowsDatatype(printerName, docName, "RAW", payload)
	}
	return 0, err
}

func isInvalidSpoolDatatypeError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "datatype is invalid") ||
		strings.Contains(msg, "invalid datatype") ||
		strings.Contains(msg, "specified datatype")
}

func rawPrintWindowsDatatype(printerName, docName, datatype string, payload []byte) (uint32, error) {
	if len(payload) == 0 {
		return 0, fmt.Errorf("print ma'lumoti bo'sh")
	}
	if strings.TrimSpace(datatype) == "" {
		datatype = "RAW"
	}

	printerNamePtr, err := windows.UTF16PtrFromString(printerName)
	if err != nil {
		return 0, err
	}

	var printerHandle windows.Handle
	ret, _, callErr := procOpenPrinterW.Call(
		uintptr(unsafe.Pointer(printerNamePtr)),
		uintptr(unsafe.Pointer(&printerHandle)),
		0,
	)
	if ret == 0 {
		return 0, fmt.Errorf("OpenPrinter: %w", callErr)
	}
	defer procClosePrinter.Call(uintptr(printerHandle))

	if docName == "" {
		docName = "AC Label V2"
	}
	docNamePtr, err := windows.UTF16PtrFromString(docName)
	if err != nil {
		return 0, err
	}
	rawTypePtr, err := windows.UTF16PtrFromString(datatype)
	if err != nil {
		return 0, err
	}

	docInfo := docInfo1{
		DocName:    docNamePtr,
		OutputFile: nil,
		Datatype:   rawTypePtr,
	}

	jobID, _, callErr := procStartDocPrinterW.Call(
		uintptr(printerHandle),
		1,
		uintptr(unsafe.Pointer(&docInfo)),
	)
	if jobID == 0 {
		return 0, fmt.Errorf("StartDocPrinter: %w", callErr)
	}
	// Safety net for the error paths below; the success path ends the document
	// explicitly so that its failure is not swallowed.
	docOpen := true
	defer func() {
		if docOpen {
			procEndDocPrinter.Call(uintptr(printerHandle))
		}
	}()

	ret, _, callErr = procStartPagePrinter.Call(uintptr(printerHandle))
	if ret == 0 {
		return 0, fmt.Errorf("StartPagePrinter: %w", callErr)
	}

	var written uint32
	ret, _, callErr = procWritePrinter.Call(
		uintptr(printerHandle),
		uintptr(unsafe.Pointer(&payload[0])),
		uintptr(len(payload)),
		uintptr(unsafe.Pointer(&written)),
	)
	if ret == 0 {
		return 0, fmt.Errorf("WritePrinter: %w", callErr)
	}
	if written != uint32(len(payload)) {
		return 0, fmt.Errorf("WritePrinter: yozildi %d/%d bayt", written, len(payload))
	}

	ret, _, callErr = procEndPagePrinter.Call(uintptr(printerHandle))
	if ret == 0 {
		return 0, fmt.Errorf("EndPagePrinter: %w", callErr)
	}

	// EndDocPrinter is where the spooler commits the job — a failure here means
	// nothing will print, so it must not be ignored.
	ret, _, callErr = procEndDocPrinter.Call(uintptr(printerHandle))
	docOpen = false
	if ret == 0 {
		return 0, fmt.Errorf("EndDocPrinter: %w", callErr)
	}

	return uint32(jobID), nil
}

// PrinterReachable opens the Windows printer handle briefly to verify the queue exists.
func PrinterReachable(printerName string) error {
	printerName = strings.TrimSpace(printerName)
	if printerName == "" {
		return fmt.Errorf("printer nomi bo'sh")
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
	procClosePrinter.Call(uintptr(printerHandle))
	return nil
}

// suppress unused import on some toolchains
var _ = syscall.Errno(0)
