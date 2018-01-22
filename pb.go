package progress

import (
	"github.com/cheggaaa/pb"
)

// UpdatePB ..
func UpdatePB(b **pb.ProgressBar, pt ProgressTracker) {

	status := pt.Status()

	if *b == nil {
		(*b) = pb.New(int(status.Total))
		(*b).Start()
	}

	(*b).Prefix(status.Stage)
	(*b).Set(int(status.Progress))
}
