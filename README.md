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

## Break

[pocketbase-v0.22.7](https://github.com/pocketbase/pocketbase/blob/master/CHANGELOG.md#v0227) have changed s3 implement, must upgrade v0.1.0 version to follow change
