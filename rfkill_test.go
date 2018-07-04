package rfkill

import (
	"encoding/binary"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestEach(t *testing.T) {
	withControlFile(t, func(f *os.File) {
		evs := []Event{{
			Idx:  1,
			Type: TypeWLAN,
			Soft: 1,
			Hard: 0,
		}, {
			Idx:  2,
			Type: TypeBluetooth,
			Soft: 0,
			Hard: 1,
		}}
		for _, ev := range evs {
			if err := binary.Write(f, endianness, ev); err != nil {
				t.Fatal(err)
			}
		}
		i := 0
		if err := Each(func(ev Event) error {
			if !reflect.DeepEqual(ev, evs[i]) {
				t.Fatalf("not equal events, got = %#v, want %#v", ev, evs[i])
			}
			i++
			return nil
		}); err != nil && err != io.EOF {
			t.Fatal(err)
		}
	})
}

func TestBlockByIdx(t *testing.T) {
	withControlFile(t, func(f *os.File) {
		if err := BlockByIdx(1, true); err != nil {
			t.Fatal(err)
		}
		var ev Event
		if err := binary.Read(f, endianness, &ev); err != nil {
			t.Fatal(err)
		}
		want := Event{
			Idx:  1,
			Soft: 1,
			Op:   OpChange,
		}
		if !reflect.DeepEqual(ev, want) {
			t.Fatalf("BlockByIdx received event = %#v, want %#v", ev, want)
		}
	})
}

func withControlFile(t *testing.T, fn func(f *os.File)) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	tmp := controlFile
	controlFile = f.Name()
	defer func() {
		controlFile = tmp
		os.Remove(f.Name())
	}()
	fn(f)
}
