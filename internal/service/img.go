package service

import (
	"github.com/cockroachdb/errors"
	"github.com/otiai10/gosseract/v2"
)

// ParseImageToTxt runs OCR over an image and returns plain text (or hOCR when
// returnTxt is false). Mapping OCR text to a structured receipt lives in the
// internal/service/receipt package.
func ParseImageToTxt(imagePath string, returnTxt bool) (string, error) {
	client := gosseract.NewClient()
	defer client.Close()
	client.Languages = []string{"hun", "eng"}

	client.SetImage(imagePath)
	if returnTxt {
		text, err := client.Text()
		if err != nil {
			return "", errors.Wrapf(err, "ocr text extraction path=%q", imagePath)
		}
		return text, nil
	}
	hocr, err := client.HOCRText()
	if err != nil {
		return "", errors.Wrapf(err, "ocr hocr extraction path=%q", imagePath)
	}
	return hocr, nil
}
