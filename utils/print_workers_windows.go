//go:build windows

package utils

import (
	"os/exec"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Kill-on-close job object: when the API process exits, child print workers die with it.
var (
	printWorkersJob     windows.Handle
	printWorkersJobOnce sync.Once
	printWorkersJobErr  error
)

func ensurePrintWorkersJob() error {
	printWorkersJobOnce.Do(func() {
		h, err := windows.CreateJobObject(nil, nil)
		if err != nil {
			printWorkersJobErr = err
			return
		}
		info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
		info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
		ok, err := windows.SetInformationJobObject(
			h,
			windows.JobObjectExtendedLimitInformation,
			uintptr(unsafe.Pointer(&info)),
			uint32(unsafe.Sizeof(info)),
		)
		if ok == 0 {
			_ = windows.CloseHandle(h)
			if err != nil {
				printWorkersJobErr = err
			} else {
				printWorkersJobErr = windows.ERROR_INVALID_PARAMETER
			}
			return
		}
		printWorkersJob = h
	})
	return printWorkersJobErr
}

func assignCmdToPrintWorkersJob(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	if err := ensurePrintWorkersJob(); err != nil || printWorkersJob == 0 {
		return
	}
	const access = windows.PROCESS_SET_QUOTA | windows.PROCESS_TERMINATE
	hProc, err := windows.OpenProcess(access, false, uint32(cmd.Process.Pid))
	if err != nil {
		return
	}
	defer windows.CloseHandle(hProc)
	_ = windows.AssignProcessToJobObject(printWorkersJob, hProc)
}

// StartPrintWorkers spawns the GDI and TSPL/ZPL worker processes up front, so
// the first print of the shift does not pay the multi-second cold start.
//
// This is an explicit call rather than an init(): the workers are launched by
// re-executing this binary, and doing that automatically meant any process
// linking this package — a test binary, a CLI — would spawn a full set of
// children on startup.
func StartPrintWorkers() {
	// Child --print-worker processes must not spawn nested workers.
	if isPrintWorkerProcess() {
		return
	}
	go gdiPool.warmUp()
	go tsplRawPool.warmUp()
	go zplRawPool.warmUp()
}

// StopAllPrintWorkers stops every GDI + TSPL/ZPL child worker (best-effort).
func StopAllPrintWorkers() {
	gdiPool.ensure()
	for _, w := range gdiPool.all {
		w.mu.Lock()
		w.stopLocked()
		w.mu.Unlock()
	}

	for _, pool := range []*rawPrintWorkerPool{tsplRawPool, zplRawPool} {
		pool.ensure()
		for _, w := range pool.all {
			w.mu.Lock()
			w.stopLocked()
			w.mu.Unlock()
		}
	}

	// Closing the job object is the backstop that kills anything still running.
	if printWorkersJob != 0 {
		_ = windows.CloseHandle(printWorkersJob)
		printWorkersJob = 0
	}
}
