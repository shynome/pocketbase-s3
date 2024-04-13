package s3

import (
	"context"
	"fmt"
	"mime"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/types"
	s3hook "github.com/shynome/pocketbase-s3/hook"
	"golang.org/x/sync/errgroup"
)

// FixObjectHeaders add ContentDisposition header, if file is protected header CacheControl: no-cache also be added
func FixObjectHeaders(app *pocketbase.PocketBase, tags ...string) {
	app.OnRecordAfterCreateRequest(tags...).Add(func(e *core.RecordCreateEvent) error {
		if len(e.UploadedFiles) == 0 {
			return nil
		}
		ctx := e.HttpContext.Request().Context()
		return fixObjectHeaders(ctx, app, e.Record, e.UploadedFiles)
	})
	app.OnRecordAfterUpdateRequest(tags...).Add(func(e *core.RecordUpdateEvent) error {
		if len(e.UploadedFiles) == 0 {
			return nil
		}
		ctx := e.HttpContext.Request().Context()
		return fixObjectHeaders(ctx, app, e.Record, e.UploadedFiles)
	})
}

func fixObjectHeaders(ctx context.Context, app *pocketbase.PocketBase, record *models.Record, filesMap map[string][]*filesystem.File) (err error) {
	defer func() {
		if err != nil {
			err = echo.NewHTTPErrorWithInternal(http.StatusInternalServerError, err, "fix s3 object file headers failed")
		}
	}()
	settings := app.Settings()
	if s3 := settings.S3.Enabled; !s3 {
		return nil
	}

	fs, err := app.NewFilesystem()
	if err != nil {
		return err
	}
	defer fs.Close()
	client := s3hook.GetClient(fs)
	bucket := settings.S3.Bucket

	fieldsSchema := record.Collection().Schema
	baseFilesPath := record.BaseFilesPath()

	eg := new(errgroup.Group)
	for field, files := range filesMap {
		fieldSchema := fieldsSchema.GetFieldByName(field)
		options := fieldSchema.Options.(*schema.FileOptions)
		var cacheControl *string
		if options.Protected {
			cacheControl = types.Pointer("no-cache")
		}
		for _, file := range files {
			originPath := baseFilesPath + "/" + file.Name
			disposition := mime.FormatMediaType("attachment", map[string]string{"filename": file.OriginalName})
			eg.Go(func() error {
				obj, err := client.GetObject(ctx, &s3.GetObjectInput{
					Bucket: &bucket,
					Key:    &originPath,
				})
				if err != nil {
					return err
				}
				if obj.ContentDisposition != nil {
					return nil
				}
				output, err := client.CopyObject(ctx, &s3.CopyObjectInput{
					Bucket:     &bucket,
					Key:        &originPath,
					CopySource: types.Pointer(fmt.Sprintf("%s/%s", bucket, originPath)),
					// add ContentDisposition header
					ContentDisposition: &disposition,
					// prototect file should not be cached
					CacheControl: cacheControl,
					// other options copy
					Metadata:        obj.Metadata,
					ContentEncoding: obj.ContentEncoding,
					ContentLanguage: obj.ContentLanguage,
					ContentType:     obj.ContentType,
				})
				if err != nil {
					return err
				}
				_ = output
				return nil
			})
		}
	}

	if err = eg.Wait(); err != nil {
		return err
	}

	return nil
}
