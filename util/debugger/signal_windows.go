package debugger

import "os"

// compareSignal is the signal to trigger node info. For windows,
// it's SIGINT.
var CompareSignal = os.Interrupt
