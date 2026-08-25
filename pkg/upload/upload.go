package upload

import (
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ajderniz/repostele/pkg/pass"
)

const _MAX_IMG_BYTES = 5 << 20 // 5 MiB

var _AllowedImgExt = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

var (
	ErrTooLarge  = errors.New("La imagen es muy grande (máx. 5MB)")
	ErrBadType   = errors.New(
		"El archivo no es un formato de imagen aceptable (jpg, png, gif, webp)",
	)
	_ErrSave     = errors.New("No se pudo guardar la imagen")
)

// SaveImage reads a multipart file, validates it is actually an image
// (by sniffing content, not trusting the client-supplied header), and
// writes it to dir with a random generated filename. Returns the path
// to the saved file relative to dir (i.e. just the filename), which the
// caller should join with the URL-facing prefix (e.g. "/dyn/items/").
func SaveImage(file multipart.File, header *multipart.FileHeader, dir string) (string, error) {
	defer file.Close()

	if header.Size > _MAX_IMG_BYTES { return "", ErrTooLarge }

	// Sniff real content type from the first 512 bytes; never trust
	// header.Header.Get("Content-Type") alone, it's client-controlled.
	sniff := make([]byte, 512)
	n, err := file.Read(sniff)
	if err != nil && err != io.EOF { slog.Error(err.Error()); return "", _ErrSave}
	contentType := http.DetectContentType(sniff[:n])

	ext, ok := _AllowedImgExt[contentType]
	if !ok { return "", ErrBadType }

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		slog.Error(err.Error())
		return "", _ErrSave
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		slog.Error(err.Error())
		return "", _ErrSave
	}

	name, err := pass.GenerateToken(16)
	if err != nil { return "", _ErrSave }
	// GenerateToken uses base64.URLEncoding, which can still contain
	// '/' and '=' — neither is filename-safe. Sanitize before use.
	name = sanitizeFilename(name)

	dst, err := os.Create(filepath.Join(dir, name+ext))
	if err != nil {
		slog.Error(err.Error())
		return "", _ErrSave
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		slog.Error(err.Error())
		os.Remove(dst.Name())
		return "", _ErrSave
	}

	return name + ext, nil
}

func sanitizeFilename(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			out = append(out, c)
		}
	}
	return string(out)
}
