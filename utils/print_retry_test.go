package utils

import (
	"errors"
	"fmt"
	"testing"
)

func TestRetryPrintV2StopsOnUnsafeError(t *testing.T) {
	// A worker reply timeout leaves it unknown whether the label came out, so
	// resubmitting could produce a duplicate.
	unsafe := errors.New("tspl worker javobi o'qilmadi: worker javob vaqti tugadi")

	calls := 0
	err := RetryPrintV2(3, func(int) error {
		calls++
		return unsafe
	})

	if calls != 1 {
		t.Errorf("urinishlar soni %d, kutilgan 1 (qayta urinmasligi kerak)", calls)
	}
	if !errors.Is(err, unsafe) {
		t.Errorf("asl xato saqlanmadi: %v", err)
	}
}

func TestRetryPrintV2RetriesBlockedSpool(t *testing.T) {
	// A blocked job is cancelled before anything printed, so resubmitting is safe.
	blocked := fmt.Errorf("%w: job #5 status=Paused", errSpoolJobBlocked)

	calls := 0
	err := RetryPrintV2(3, func(attempt int) error {
		calls++
		if attempt < 3 {
			return blocked
		}
		return nil
	})

	if err != nil {
		t.Fatalf("uchinchi urinishda muvaffaqiyat kutilgan, olindi: %v", err)
	}
	if calls != 3 {
		t.Errorf("urinishlar soni %d, kutilgan 3", calls)
	}
}

func TestRetryPrintV2RetriesUnavailableWorker(t *testing.T) {
	unavailable := fmt.Errorf("%w: jarayon o'lgan", errRawWorkerUnavailable)

	calls := 0
	err := RetryPrintV2(2, func(int) error {
		calls++
		return unavailable
	})

	if calls != 2 {
		t.Errorf("urinishlar soni %d, kutilgan 2", calls)
	}
	if !errors.Is(err, errRawWorkerUnavailable) {
		t.Errorf("xato sababi saqlanmadi: %v", err)
	}
}

func TestIsRetryablePrintError(t *testing.T) {
	retryable := []error{
		fmt.Errorf("spool: %w: job #1 status=Offline", errSpoolJobBlocked),
		fmt.Errorf("%w: worker start", errRawWorkerUnavailable),
	}
	for _, err := range retryable {
		if !isRetryablePrintError(err) {
			t.Errorf("isRetryablePrintError(%v) = false, kutilgan true", err)
		}
	}

	notRetryable := []error{
		errors.New("gdi: printerga yuborishda xatolik"),
		errors.New("worker javob vaqti tugadi"),
		errors.New("render: shrift topilmadi"),
		nil,
	}
	for _, err := range notRetryable {
		if isRetryablePrintError(err) {
			t.Errorf("isRetryablePrintError(%v) = true, kutilgan false", err)
		}
	}
}
