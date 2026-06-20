package service

import (
	"github.com/otiai10/gosseract/v2"
)

func ParseImageToTxt(imagePath string, returnTxt bool) (string, error) {
	client := gosseract.NewClient()
	defer client.Close()
	client.Languages = []string{"eng"}

	client.SetImage(imagePath)
	if returnTxt {
		text, err := client.Text()
		if err != nil {
			return text, err
		}
		return text, nil
	}
	hocr, err := client.HOCRText()
	if err != nil {
		return hocr, err
	}
	return hocr, nil
}
