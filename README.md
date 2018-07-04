# rfkill

A golang client for [rfkill](https://github.com/torvalds/linux/blob/master/Documentation/rfkill.txt). It supports reading available rfkill devices, subscribing to events and blocking/unblocking devices.

## Usage

```go
package main

import (
	"fmt"
	"os"

	"github.com/goautomotive/rfkill"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	return rfkill.Each(func(ev rfkill.Event) error {
		if ev.Soft == 0 {
			return nil
		}
		name, err := rfkill.NameByIdx(ev.Idx)
		if err != nil {
			return err
		}
		fmt.Printf("unblocking: %s\n", name)
		return rfkill.BlockByIdx(ev.Idx, false)
	})
}
```
