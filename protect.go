package s3

import (
	"context"
	"net/http"

	"github.com/aws/aws-sdk-go/service/s3"
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

func ProtectFile(app *pocketbase.PocketBase, tags ...string) {
	app.OnRecordAfterCreateRequest(tags...).Add(func(e *core.RecordCreateEvent) error {
		if len(e.UploadedFiles) == 0 {
			return nil
		}
		ctx := e.HttpContext.Request().Context()
		return protectFile(ctx, app, e.Record, e.UploadedFiles)
	})
	app.OnRecordAfterUpdateRequest(tags...).Add(func(e *core.RecordUpdateEvent) error {
		if len(e.UploadedFiles) == 0 {
			return nil
		}
		ctx := e.HttpContext.Request().Context()
		return protectFile(ctx, app, e.Record, e.UploadedFiles)
	})
}

func protectFile(ctx context.Context, app *pocketbase.PocketBase, record *models.Record, filesMap map[string][]*filesystem.File) (err error) {
	defer func() {
		if err != nil {
			err = echo.NewHTTPErrorWithInternal(http.StatusInternalServerError, err, "protected file set s3 acl private failed")
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
		if !options.Protected {
			continue
		}
		for _, file := range files {
			originPath := baseFilesPath + "/" + file.Name
			eg.Go(func() error {
				output, err := client.PutObjectAclWithContext(ctx, &s3.PutObjectAclInput{
					Bucket: &bucket,
					Key:    &originPath,
					ACL:    types.Pointer("private"),
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
