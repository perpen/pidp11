package pidp11

import "syscall"

func nanosleep(ns int) {
	ts := syscall.Timespec{Sec: 0, Nsec: int64(ns)}
	syscall.Nanosleep(&ts, nil)
}
