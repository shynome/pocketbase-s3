## How to Use

```go
package main

import (
	"github.com/pocketbase/pocketbase"
	s3 "github.com/shynome/pocketbase-s3"
)

func initApp(app *pocketbase.PocketBase) {
	s3.FixObjectHeaders(app)
	s3.ProtectFile(app)
}

```
