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
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

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

	controlFile = f.Name()

	i := 0
	if err = Each(func(ev Event) error {
		if !reflect.DeepEqual(ev, evs[i]) {
			t.Fatalf("not equal events, got = %v, want %v", ev, evs[i])
		}
		i++
		return nil
	}); err != nil && err != io.EOF {
		t.Fatal(err)
	}
}
