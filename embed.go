//go:build embedded

package orchestra

import "embed"

//go:embed static/assets
//go:embed static/index.html
//go:embed static/main.wasm
var embedded embed.FS
