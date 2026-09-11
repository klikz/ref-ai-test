//go:build !windows

package utils

import "fmt"

func rawPrintViaWorker(language, printerName, docName, datatype string, payload []byte) (uint32, error) {
	_ = language
	_ = printerName
	_ = docName
	_ = datatype
	_ = payload
	return 0, fmt.Errorf("RAW print worker faqat Windowsda mavjud")
}
