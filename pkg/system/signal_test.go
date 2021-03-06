package system

import (
	"os"
	"syscall"
	"testing"

	"github.com/sirupsen/logrus"
)

const (
	testSignal                        = syscall.SIGUSR2
	customSignalType       SignalType = 777
	customSignalTypeString            = "777"
)

// testHandling implement simplest Operator interface
type testHandling struct {
	ch chan SignalType
}

// Reload implementation
func (th testHandling) Reload() error {
	th.ch <- Reload
	return nil
}

// Maintenance implementation
func (th testHandling) Maintenance() error {
	th.ch <- Maintenance
	return nil
}

// Shutdown implementation
func (th testHandling) Shutdown() []error {
	var errs []error
	th.ch <- Shutdown
	return errs
}

func TestSignals(t *testing.T) {
	// Setup logger
	pid := os.Getpid()
	proc, err := os.FindProcess(pid)
	if err != nil {
		t.Error("Finding process:", err)
	}

	signals := NewSignals(make(chan error))

	shutdownSignals := signals.Get(Shutdown)
	verifySignal(t, syscall.SIGTERM, shutdownSignals, Shutdown)
	verifySignal(t, syscall.SIGINT, shutdownSignals, Shutdown)
	reloadSignals := signals.Get(Reload)
	verifySignal(t, syscall.SIGHUP, reloadSignals, Reload)
	maintenanceSignals := signals.Get(Maintenance)
	verifySignal(t, syscall.SIGUSR1, maintenanceSignals, Maintenance)

	handling := &testHandling{ch: make(chan SignalType, 1)}
	go func() {
		if err := signals.Wait(logrus.WithField("component", "test"), handling); err != nil {
			t.Error("Waiting signal:", err)
		}
	}()

	// Prepare and send reload signal
	signals.Add(testSignal, Reload)
	sendSignal(t, handling.ch, proc, Reload)
	signals.Remove(testSignal, Reload)

	// Prepare and send maintenance signal
	signals.Add(testSignal, Maintenance)
	sendSignal(t, handling.ch, proc, Maintenance)
	signals.Remove(testSignal, Maintenance)

	// Prepare and send shutdown signal
	signals.Add(testSignal, Shutdown)
	sendSignal(t, handling.ch, proc, Shutdown)
	signals.Remove(testSignal, Shutdown)
}

func sendSignal(t *testing.T, ch <-chan SignalType, proc *os.Process, signal SignalType) {
	err := proc.Signal(testSignal)
	if err != nil {
		t.Error("Sending signal:", err)
		return
	}
	if sig := <-ch; sig != signal {
		t.Error("Expected signal:", signal, "got", sig)
	}
}

func verifySignal(t *testing.T, signal os.Signal, signals []os.Signal, sigType SignalType) {
	if !isSignalAvailable(signal, signals) {
		t.Error("Absent of the signal:", signal, "among", sigType, "signal type")
	}
}

func TestSignalStringer(t *testing.T) {
	s := Shutdown
	if s.String() != "SHUTDOWN" {
		t.Error("Expected signal type SHUTDOWN, got", s.String())
	}
	s = Reload
	if s.String() != "RELOAD" {
		t.Error("Expected signal type RELOAD, got", s.String())
	}
	s = Maintenance
	if s.String() != "MAINTENANCE" {
		t.Error("Expected signal type MAINTENANCE, got", s.String())
	}
	s = customSignalType
	if s.String() != customSignalTypeString {
		t.Error("Expected signal type ", customSignalTypeString, "got", s.String())
	}
}

func TestRemoveNotExistingSignal(t *testing.T) {
	s := NewSignals(make(chan error))
	count := len(s.maintenance)
	s.Remove(syscall.SIGUSR2, Maintenance)
	if len(s.maintenance) != count {
		t.Error("Expected count of signals", count, "got", len(s.maintenance))
	}
}
