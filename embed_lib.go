//go:build !embedded

package orchestra

import "os"

var embedded = os.DirFS(`/dev/null`)
