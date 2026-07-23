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

func rawPrintWindows(printerName, docName, datatype string, payload []byte) error {
	err := rawPrintWindowsDatatype(printerName, docName, datatype, payload)
	if err == nil {
		return nil
	}
	// Some drivers (e.g. Zebra ZT411) reject "ZPL" but accept RAW passthrough.
	if datatype != "RAW" && isInvalidSpoolDatatypeError(err) {
		return rawPrintWindowsDatatype(printerName, docName, "RAW", payload)
	}
	return err
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

func rawPrintWindowsDatatype(printerName, docName, datatype string, payload []byte) error {
	if len(payload) == 0 {
		return fmt.Errorf("print ma'lumoti bo'sh")
	}
	if strings.TrimSpace(datatype) == "" {
		datatype = "RAW"
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

	if docName == "" {
		docName = "AC Label V2"
	}
	docNamePtr, err := windows.UTF16PtrFromString(docName)
	if err != nil {
		return err
	}
	rawTypePtr, err := windows.UTF16PtrFromString(datatype)
	if err != nil {
		return err
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
		return fmt.Errorf("StartDocPrinter: %w", callErr)
	}
	defer procEndDocPrinter.Call(uintptr(printerHandle))

	ret, _, callErr = procStartPagePrinter.Call(uintptr(printerHandle))
	if ret == 0 {
		return fmt.Errorf("StartPagePrinter: %w", callErr)
	}

	var written uint32
	ret, _, callErr = procWritePrinter.Call(
		uintptr(printerHandle),
		uintptr(unsafe.Pointer(&payload[0])),
		uintptr(len(payload)),
		uintptr(unsafe.Pointer(&written)),
	)
	if ret == 0 {
		return fmt.Errorf("WritePrinter: %w", callErr)
	}
	if written != uint32(len(payload)) {
		return fmt.Errorf("WritePrinter: yozildi %d/%d bayt", written, len(payload))
	}

	ret, _, callErr = procEndPagePrinter.Call(uintptr(printerHandle))
	if ret == 0 {
		return fmt.Errorf("EndPagePrinter: %w", callErr)
	}

	return nil
}

// suppress unused import on some toolchains
var _ = syscall.Errno(0)
