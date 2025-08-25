package loop

import "errors"

// ErrFileGone indicates a file was removed or disappeared during processing
var ErrFileGone = errors.New("file gone")