# rfkill

A golang client for [rfkill](https://github.com/torvalds/linux/blob/master/Documentation/rfkill.txt). It supports reading available rfkill devices, subscribing to events and blocking/unblocking devices.

## Documentation

Detailed API documentation can be found [here](https://godoc.org/github.com/amenzhinsky/rfkill).

## Usage

```go
package main

import (
	"fmt"
	"os"

	"github.com/amenzhinsky/rfkill"
)

func main() {
	if err := rfkill.Each(func(ev rfkill.Event) error {
		if ev.Soft == 0 {
			return nil
		}
		name, err := rfkill.NameByIdx(ev.Idx)
		if err != nil {
			return err
		}
		fmt.Printf("unblocking: %s\n", name)
		return rfkill.BlockByIdx(ev.Idx, false)
	}); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
```
