package profilepic

import (
	"bytes"
	"image"
	"io"

	"github.com/nfnt/resize"

	_ "image/gif"
	"image/jpeg"
	_ "image/png"
)

func MakeImage(img io.Reader) ([]byte, error) {
	orig, _, err := image.Decode(img)
	if err != nil {
		return nil, err // maybe sentinel to point out there was a problem with the image.
	}

	thumb := resize.Thumbnail(100, 100, orig, resize.Bicubic)

	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, thumb, nil)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
