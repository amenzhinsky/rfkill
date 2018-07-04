//+build linux

// This is a rfkill client library for golang, works only on linux.
//
// For implementation details see:
// https://github.com/torvalds/linux/blob/master/include/uapi/linux/rfkill.h
package rfkill

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"
	"unsafe"
)

// Op is operation type.
type Op uint8

const (
	// OpAdd a device is added.
	OpAdd = iota

	// OpDel a device is deleted.
	OpDel

	// OpChange a device's state is changed.
	OpChange

	// OpChangeAll userspace changes in all devices.
	OpChangeAll
)

func (op Op) String() string {
	switch op {
	case OpAdd:
		return "add"
	case OpDel:
		return "delete"
	case OpChange:
		return "change"
	case OpChangeAll:
		return "change-all"
	default:
		return ""
	}
}

// Type is type of rfkill switch.
type Type uint8

const (
	// TypeAll toggles all switches, useless in this library.
	TypeAll = iota

	// TypeWLAN switch is on a 802.11 wireless network device.
	TypeWLAN

	// TypeBluetooth switch is on a bluetooth device.
	TypeBluetooth

	// TypeUWB switch is on a ultra wideband device.
	TypeUWB

	// TypeWiMAX switch is on a WiMAX device.
	TypeWiMAX

	// TypeWWAN switch is on a wireless WAN device.
	TypeWWAN

	// TypeGPS switch is on a GPS device.
	TypeGPS

	// TypeFM switch is on a FM radio device.
	TypeFM

	// TypeNFC switch is on an NFC device.
	TypeNFC
)

func (typ Type) String() string {
	switch typ {
	case TypeAll:
		return "all"
	case TypeWLAN:
		return "wifi"
	case TypeBluetooth:
		return "bluetooth"
	case TypeUWB:
		return "uwb"
	case TypeWiMAX:
		return "wimax"
	case TypeWWAN:
		return "wwan"
	case TypeGPS:
		return "gps"
	case TypeFM:
		return "fm"
	case TypeNFC:
		return "nfc"
	default:
		return ""
	}
}

// NameByIdx returns system name for the named device idx.
func NameByIdx(idx uint32) (string, error) {
	b, err := ioutil.ReadFile(fmt.Sprintf("/sys/class/rfkill/rfkill%d/name", idx))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Event is a rfkill event read from /dev/rfkill.
type Event struct {
	// Idx is device index.
	Idx uint32

	// Type of the event.
	Type Type

	// Op operation code.
	Op Op

	// Soft state.
	Soft uint8

	// Hard state.
	Hard uint8
}

var endianness binary.ByteOrder = binary.LittleEndian

func init() {
	b := [2]byte{0x0, 0x1}
	if *(*uint16)(unsafe.Pointer(&b[0])) == 1 {
		endianness = binary.BigEndian
	}
}

// BlockByIdx soft blocks or unblocks a device by the given idx.
func BlockByIdx(idx uint32, block bool) error {
	f, err := open(os.O_WRONLY)
	if err != nil {
		return err
	}
	defer f.Close()

	var soft uint8
	if block {
		soft = 1
	}
	return binary.Write(f, endianness, &Event{
		Idx:  idx,
		Op:   OpChange,
		Soft: soft,
	})
}

// Each iterates over all registered devices yielding them as OpAdd events.
// If fn returns an error the function immediately propagates it.
//
// Example how to unblock all devices:
//
// 	if err := rfkill.Each(func(ev rfkill.Event) error {
// 		return rfkill.BlockByIdx(ev.Idx, false)
// 	}); err != nil {
// 		return err
// 	}
func Each(fn func(ev Event) error) error {
	w, err := Watch(OpAdd)
	if err != nil {
		return err
	}
	defer w.Close()

	for {
		select {
		case ev, ok := <-w.C():
			if !ok {
				return w.Err()
			}
			if err = fn(ev); err != nil {
				return err
			}
			// it emulates the EAGAIN error
		case <-time.After(time.Millisecond):
			return nil
		}
	}
}

// Watch monitors the rfkill events.
//
// If ops is not empty it acts as a filter, otherwise it delivers everything.
//
// Example:
// 	w, err := rfkill.Watch()
// 	if err != nil {
// 		return err
// 	}
// 	defer w.Close()
//
// 	for ev := range w.C() {
// 		fmt.Printf("idx=%d type=%s soft=%t hard=%t",
// 			ev.Idx, ev.Type, ev.Soft != 0, ev.Hard != 0)
// 	}
// 	if err = w.Err(); err != nil {
// 		return err
// 	}
func Watch(ops ...Op) (*Watcher, error) {
	f, err := open(os.O_RDONLY)
	if err != nil {
		return nil, err
	}
	w := &Watcher{
		file: f,
		evch: make(chan Event),
		done: make(chan struct{}),
	}
	go w.watch(ops)
	return w, nil
}

// Watcher is a event watching instance.
type Watcher struct {
	err  error
	file *os.File
	evch chan Event
	done chan struct{}
}

// ErrClosed denotes closed watcher.
var ErrClosed = errors.New("rfkill: closed")

func (w *Watcher) watch(ops []Op) {
	defer close(w.evch)

	var ev Event
	for {
		if err := binary.Read(w.file, endianness, &ev); err != nil {
			if e, ok := err.(*os.PathError); ok && e.Timeout() {
				return // Close caused this, ignore
			}
			w.close(err)
			return
		}
		if len(ops) != 0 {
			var found bool
			for _, op := range ops {
				if op == ev.Op {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		select {
		case w.evch <- ev:
		case <-w.done:
			return
		}
	}
}

// C is a rfkill events stream.
func (w *Watcher) C() <-chan Event {
	return w.evch
}

// Err is the watcher's error, it makes sense to call it only after
// the channel returned from C gets closed.
func (w *Watcher) Err() error {
	return w.err
}

// Close makes the watcher to stop automatically closing the events stream channel.
func (w *Watcher) Close() error {
	return w.close(ErrClosed)
}

func (w *Watcher) close(err error) error {
	select {
	case <-w.done:
		return nil
	default:
	}

	// golang abstracts nonblocking read in the runtime, the only
	// way to work this around is set a read timeout from the past
	w.err = err
	w.file.SetReadDeadline(time.Now())
	close(w.done)
	return w.file.Close()
}

// not a constant for testing purposes.
var controlFile = "/dev/rfkill"

func open(flags int) (*os.File, error) {
	f, err := os.OpenFile(controlFile, flags, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("rfkill: control device is missing")
		}
		return nil, err
	}
	return f, nil
}
