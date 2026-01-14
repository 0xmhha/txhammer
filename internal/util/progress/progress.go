package progress

import (
	"log"

	"github.com/schollz/progressbar/v3"
)

// Add increments the progress bar while safely handling errors.
func Add(bar *progressbar.ProgressBar, n int) {
	if bar == nil || n == 0 {
		return
	}

	if err := bar.Add(n); err != nil {
		log.Printf("failed to update progress bar: %v", err)
	}
}
