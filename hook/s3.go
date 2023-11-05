package hook

import (
	"context"
	"unsafe"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"gocloud.dev/blob"
)

func GetClient(fs *filesystem.System) *s3.S3 {
	b := GetBucket(fs)
	var x *s3.S3
	if ok := b.As(&x); !ok {
		panic("get *s3.S3 failed")
	}
	return x
}

func GetBucket(fs *filesystem.System) *blob.Bucket {
	x := *(*system)(unsafe.Pointer(fs))
	return x.bucket
}

type system struct {
	ctx    context.Context
	bucket *blob.Bucket
}
