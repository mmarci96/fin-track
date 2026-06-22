package img

import (
	"github.com/cockroachdb/errors"
	"github.com/otiai10/gosseract/v2"
)

type ImgService struct {
	client *gosseract.Client
}

func NewImgService() *ImgService {
	client := gosseract.NewClient()
	defer client.Close()
	client.Languages = []string{"hun", "eng"}
	return &ImgService{client: client}
}

// Parse img to text with gossreract client.HOCRText and returns the results
func GetHOCRText(imgPath string) (string, error) {
	client := gosseract.NewClient()
	defer client.Close()
	client.Languages = []string{"hun", "eng"}
	client.SetImage(imgPath)
	hocr, err := client.HOCRText()
	if err != nil {
		return "", errors.Wrapf(err, "ocr hocr extraction path=%q", imgPath)
	}
	return hocr, nil
}

// ParseImageToTxt runs OCR over an image and returns text. For the text path it
// reconstructs layout from word bounding boxes so column gaps (notably the
// far-right price column) survive as tab separators instead of collapsing into
// noise — e.g. "Maretti 70 g  399" stays separable rather than becoming "9399".
// It falls back to plain client.Text() if box data is unavailable.
func ParseImageToTxt(imagePath string, returnTxt bool) (string, error) {
	client := gosseract.NewClient()
	defer client.Close()
	client.Languages = []string{"hun", "eng"}
	client.SetImage(imagePath)

	if !returnTxt {
		hocr, err := client.HOCRText()
		if err != nil {
			return "", errors.Wrapf(err, "ocr hocr extraction path=%q", imagePath)
		}
		return hocr, nil
	}

	boxes, err := client.GetBoundingBoxesVerbose()
	if err == nil && len(boxes) > 0 {
		return reconstructLayout(boxesToWords(boxes)), nil
	}

	text, terr := client.Text()
	if terr != nil {
		return "", errors.Wrapf(terr, "ocr text extraction path=%q", imagePath)
	}
	return text, nil
}
