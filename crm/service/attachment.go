package service

import (
	"bytes"
	"context"
	"image"
	"image/gif"
	"io"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/edwvee/exiffix"
	"github.com/pkg/errors"
	"github.com/titpetric/factory"

	"github.com/crusttech/crust/crm/repository"
	"github.com/crusttech/crust/crm/types"
	"github.com/crusttech/crust/internal/auth"
	"github.com/crusttech/crust/internal/store"
	systemService "github.com/crusttech/crust/system/service"
)

const (
	attachmentPreviewMaxWidth  = 320
	attachmentPreviewMaxHeight = 180
)

type (
	attachment struct {
		db  *factory.DB
		ctx context.Context

		store store.Store
		usr   systemService.UserService

		attachment repository.AttachmentRepository
	}

	AttachmentService interface {
		With(ctx context.Context) AttachmentService

		FindByID(id uint64) (*types.Attachment, error)
		Find(filter types.AttachmentFilter) (types.AttachmentSet, types.AttachmentFilter, error)
		CreatePageAttachment(name string, size int64, fh io.ReadSeeker, pageID uint64) (*types.Attachment, error)
		CreateRecordAttachment(name string, size int64, fh io.ReadSeeker, moduleID, recordID uint64, fieldName string) (*types.Attachment, error)
		OpenOriginal(att *types.Attachment) (io.ReadSeeker, error)
		OpenPreview(att *types.Attachment) (io.ReadSeeker, error)
	}
)

func Attachment(store store.Store) AttachmentService {
	return (&attachment{
		store: store,
		usr:   systemService.DefaultUser,
	}).With(context.Background())
}

func (svc *attachment) With(ctx context.Context) AttachmentService {
	db := repository.DB(ctx)
	return &attachment{
		db:  db,
		ctx: ctx,

		store: svc.store,
		usr:   svc.usr.With(ctx),

		attachment: repository.Attachment(ctx, db),
	}
}

func (svc *attachment) FindByID(id uint64) (*types.Attachment, error) {
	// @todo [SECURITY] check if record/page can be accessed
	return svc.attachment.FindByID(id)
}

func (svc *attachment) Find(filter types.AttachmentFilter) (types.AttachmentSet, types.AttachmentFilter, error) {
	// @todo [SECURITY] enforce filter combination (page / module+record+field) & check access
	return svc.attachment.Find(filter)
}

func (svc *attachment) OpenOriginal(att *types.Attachment) (io.ReadSeeker, error) {
	if len(att.Url) == 0 {
		return nil, nil
	}

	return svc.store.Open(att.Url)
}

func (svc *attachment) OpenPreview(att *types.Attachment) (io.ReadSeeker, error) {
	if len(att.PreviewUrl) == 0 {
		return nil, nil
	}

	return svc.store.Open(att.PreviewUrl)
}

func (svc *attachment) CreatePageAttachment(name string, size int64, fh io.ReadSeeker, pageID uint64) (*types.Attachment, error) {
	var currentUserID uint64 = auth.GetIdentityFromContext(svc.ctx).Identity()

	// @todo verify if current user can access this page
	// @todo verify if current user can upload to this page

	att := &types.Attachment{
		ID:      factory.Sonyflake.NextID(),
		OwnerID: currentUserID,
		Name:    strings.TrimSpace(name),
		Kind:    types.PageAttachment,
	}

	return att, svc.create(name, size, fh, att)
}
func (svc *attachment) CreateRecordAttachment(name string, size int64, fh io.ReadSeeker, moduleID, recordID uint64, fieldName string) (*types.Attachment, error) {
	var currentUserID uint64 = auth.GetIdentityFromContext(svc.ctx).Identity()

	// @todo verify if current user can access this record
	// @todo verify if current user can upload to this record

	att := &types.Attachment{
		ID:      factory.Sonyflake.NextID(),
		OwnerID: currentUserID,
		Name:    strings.TrimSpace(name),
		Kind:    types.RecordAttachment,
	}

	return att, svc.create(name, size, fh, att)
}

func (svc *attachment) create(name string, size int64, fh io.ReadSeeker, att *types.Attachment) (err error) {
	if svc.store == nil {
		return errors.New("Can not create attachment: store handler not set")
	}

	// Extract extension but make sure path.Ext is not confused by any leading/trailing dots
	att.Meta.Original.Extension = strings.Trim(path.Ext(strings.Trim(name, ".")), ".")

	att.Meta.Original.Size = size
	if att.Meta.Original.Mimetype, err = svc.extractMimetype(fh); err != nil {
		return
	}

	log.Printf(
		"Processing uploaded file (name: %s, size: %d, mimetype: %s)",
		att.Name,
		att.Meta.Original.Size,
		att.Meta.Original.Mimetype)

	att.Url = svc.store.Original(att.ID, att.Meta.Original.Extension)
	if err = svc.store.Save(att.Url, fh); err != nil {
		log.Print(err.Error())
		return
	}

	// Process image: extract width, height, make preview
	log.Printf("Image processed, error: %v", svc.processImage(fh, att))

	log.Printf("File %s stored as %s", att.Name, att.Url)

	return svc.db.Transaction(func() (err error) {
		if att, err = svc.attachment.Create(att); err != nil {
			return
		}

		return nil
	})
}

func (svc *attachment) extractMimetype(file io.ReadSeeker) (mimetype string, err error) {
	if _, err = file.Seek(0, 0); err != nil {
		return
	}

	// Make sure we rewind when we're done
	defer file.Seek(0, 0)

	// See http.DetectContentType about 512 bytes
	var buf = make([]byte, 512)
	if _, err = file.Read(buf); err != nil {
		return
	}

	return http.DetectContentType(buf), nil
}

func (svc *attachment) processImage(original io.ReadSeeker, att *types.Attachment) (err error) {
	if !strings.HasPrefix(att.Meta.Original.Mimetype, "image/") {
		// Only supporting previews from images (for now)
		return
	}

	var (
		preview       image.Image
		opts          []imaging.EncodeOption
		format        imaging.Format
		previewFormat imaging.Format
		animated      bool
		f2m           = map[imaging.Format]string{
			imaging.JPEG: "image/jpeg",
			imaging.GIF:  "image/gif",
		}

		f2e = map[imaging.Format]string{
			imaging.JPEG: "jpg",
			imaging.GIF:  "gif",
		}
	)

	if _, err = original.Seek(0, 0); err != nil {
		return
	}

	if format, err = imaging.FormatFromExtension(att.Meta.Original.Extension); err != nil {
		return errors.Wrapf(err, "Could not get format from extension '%s'", att.Meta.Original.Extension)
	}

	previewFormat = format

	if imaging.JPEG == format {
		// Rotate image if needed
		if preview, _, err = exiffix.Decode(original); err != nil {
			//return errors.Wrapf(err, "Could not decode EXIF from JPEG")
		}

	}

	if imaging.GIF == format {
		// Decode all and check loops & delay to determine if GIF is animated or not
		if cfg, err := gif.DecodeAll(original); err == nil {
			animated = cfg.LoopCount > 0 || len(cfg.Delay) > 1

			// Use first image for the preview
			preview = cfg.Image[0]
		} else {
			return errors.Wrapf(err, "Could not decode gif config")
		}

	} else {
		// Use GIF preview for GIFs and JPEG for everything else!
		previewFormat = imaging.JPEG

		// Store with a bit lower quality
		opts = append(opts, imaging.JPEGQuality(85))
	}

	// In case of JPEG we decode the image and rotate it beforehand
	// other cases are handled here
	if preview == nil {
		if preview, err = imaging.Decode(original); err != nil {
			return errors.Wrapf(err, "Could not decode original image")
		}
	}

	var width, height = preview.Bounds().Max.X, preview.Bounds().Max.Y
	att.SetOriginalImageMeta(width, height, animated)

	if width > attachmentPreviewMaxWidth && width > height {
		// Landscape does not fit
		preview = imaging.Resize(preview, attachmentPreviewMaxWidth, 0, imaging.Lanczos)
	} else if height > attachmentPreviewMaxHeight {
		// Height does not fit
		preview = imaging.Resize(preview, 0, attachmentPreviewMaxHeight, imaging.Lanczos)
	}

	// Get dimensions from the preview
	width, height = preview.Bounds().Max.X, preview.Bounds().Max.Y

	log.Printf("Generated preview %s (%dx%dpx)", previewFormat, width, height)

	var buf = &bytes.Buffer{}
	if err = imaging.Encode(buf, preview, previewFormat); err != nil {
		return
	}

	meta := att.SetPreviewImageMeta(width, height, false)
	meta.Size = int64(buf.Len())
	meta.Mimetype = f2m[previewFormat]
	meta.Extension = f2e[previewFormat]

	// Can and how we make a preview of this attachment?
	att.PreviewUrl = svc.store.Preview(att.ID, meta.Extension)

	return svc.store.Save(att.PreviewUrl, buf)
}

var _ AttachmentService = &attachment{}
