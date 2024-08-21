//go:build avrwatchdog

package machine

import (
	"device/avr"
	"errors"
	"runtime/interrupt"
)

type wtchdg struct {
	flags uint8
}

func (w *wtchdg) Configure(config WatchdogConfig) error {
	if config.TimeoutMillis <= 16 {
		w.flags = 0b000000
	} else if config.TimeoutMillis <= 32 {
		w.flags = 0b000001
	} else if config.TimeoutMillis <= 64 {
		w.flags = 0b000010
	} else if config.TimeoutMillis <= 128 {
		w.flags = 0b000011
	} else if config.TimeoutMillis <= 256 {
		w.flags = 0b000100
	} else if config.TimeoutMillis <= 512 {
		w.flags = 0b000101
	} else if config.TimeoutMillis <= 1024 {
		w.flags = 0b000110
	} else if config.TimeoutMillis <= 2048 {
		w.flags = 0b000111
	} else if config.TimeoutMillis <= 4096 {
		w.flags = 0b100000
	} else if config.TimeoutMillis <= 8192 {
		w.flags = 0b100001
	} else {
		return errors.New("too big timeout; expected under or equal to 8192 ms")
	}
	w.flags |= avr.WDTCSR_WDE
	return nil
}

func (w *wtchdg) DisableWatchdog() {
	var state = interrupt.Disable()
	avr.Asm("wdr")
	avr.MCUSR.Set(avr.MCUSR.Get() & ^uint8(1<<3))
	avr.WDTCSR.Set(avr.WDTCSR.Get() | avr.WDTCSR_WDCE | avr.WDTCSR_WDE)
	avr.WDTCSR.Set(0)
	interrupt.Restore(state)
}

func (w *wtchdg) Start() error {
	w.DisableWatchdog()
	var state = interrupt.Disable()
	avr.Asm("wdr")
	avr.WDTCSR.Set(avr.WDTCSR.Get() | avr.WDTCSR_WDCE | avr.WDTCSR_WDE)
	avr.WDTCSR.Set(w.flags)
	interrupt.Restore(state)
	return nil
}

func (w *wtchdg) Update() {
	avr.Asm("wdr")
}

var Watchdog watchdog = watchdog(&wtchdg{})

const WatchdogMaxTimeout = 8192
