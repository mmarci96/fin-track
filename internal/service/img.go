package service

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/agnivade/levenshtein"
	"github.com/google/uuid"
	"github.com/mmarci96/fin-track/internal/model"
	"github.com/otiai10/gosseract/v2"
)

var idRegex = regexp.MustCompile(`\d+-\d+-\d+`)
var longNumberRegex = regexp.MustCompile(`\d{10,}`)
var endPriceRegex = regexp.MustCompile(`(\d[\d\s]*)$`)

func similarity(a, b string) float64 {
	dist := levenshtein.ComputeDistance(a, b)
	maxLen := max(len(b), len(a))
	return 1 - float64(dist)/float64(maxLen)
}

type Match struct {
	Key   string
	Score float64
}

func findBestMatch(target string, from []string) string {
	var matches []Match
	for _, src := range from {
		s := similarity(target, src)
		match := Match{src, s}
		if len(matches) > 0 && matches[0].Score < s {
			matches[0] = match
			continue
		}
		if s > 0.85 {
			matches = append(matches, match)
		}
	}
	return matches[0].Key
}

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

func MapReceiptTxt(text string, merchants []string) (model.Receipt, error) {
	println("Mapping...")
	var receipt model.Receipt

	t := strings.Split(text, "\n")
	// merchants := repository.FindMerchants()
	var match string
	if len(merchants) > 0 {
		match = findBestMatch(t[0], merchants)
	}
	if match == "" {
		return receipt, fmt.Errorf("No Merchant matches from db")
	}
	if match == "ALDI MAGYARORSZAG ELELMISZER Bt." {
		receipt, err := mapAldiReceipt(t)
		if err != nil {
			return receipt, err
		}
	}

	// items := strings.Split(text, "\n")
	// fmt.Println("--------------------------------------------------------------------")
	// for i, row := range items {
	// 	fmt.Printf("%v. %s\n", i, row)
	// }
	// fmt.Println("--------------------------------------------------------------------")
	//
	return receipt, nil
}

func mapAldiReceipt(rows []string) (r model.Receipt, err error) {
	m := model.Merchant{Name: rows[0]}
	r.Merchant = m
	r.ID = uuid.NewString()

	for _, row := range rows[6:] {
		println(row)
		sum, sucess := strings.CutPrefix(row, "OSSZEG")
		if sucess {
			r.ScannedAmount = sum
			continue
		}
		if longNumberRegex.MatchString(row) || idRegex.MatchString(row) {
			continue
		}
		m := endPriceRegex.FindStringSubmatch(row)
		if len(m) < 2 {
			continue
		}

		price, err := strconv.Atoi(strings.ReplaceAll(m[1], " ", ""))
		if err != nil {
			continue
		}
		item, _ := strings.CutSuffix(row, m[1])
		p := model.Product{}
		p.Name = item
		p.Price = price
		r.Products = append(r.Products, p)
		r.TotalAmount += price

	}
	return r, nil
}
