package pdf

import (
	"bytes"
	"context"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"github.com/boombuler/barcode"
	qr "github.com/boombuler/barcode/qr"
	"github.com/phpdave11/gofpdf"
)

type CouponPrintData struct {
	Code string
	URL  string
}

type Service struct{}

func (s *Service) GenerateCoupons(ctx context.Context, outputPath string, redeemBaseURL string, tokens []CouponPrintData) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}
	p := gofpdf.New("P", "mm", "A4", "")
	p.SetMargins(10, 10, 10)
	// The sheet is laid out explicitly as ten rows of two coupons. Text
	// near the bottom must not trigger gofpdf's automatic page breaks.
	p.SetAutoPageBreak(false, 0)
	cellW, cellH := 95.0, 27.7
	for i, token := range tokens {
		if i%20 == 0 {
			p.AddPage()
			p.SetDrawColor(150, 150, 150)
			p.SetLineWidth(0.15)
			for row := 0; row <= 10; row++ {
				y := 10 + float64(row)*cellH
				p.Line(7, y, 10, y)
				p.Line(102, y, 108, y)
				p.Line(105, y-3, 105, y+3)
				p.Line(200, y, 203, y)
			}
		}
		row := (i % 20) / 2
		col := i % 2
		x := 10.0 + float64(col)*cellW
		y := 10.0 + float64(row)*cellH
		qrCode, err := qr.Encode(token.URL, qr.L, qr.Auto)
		if err != nil {
			return err
		}
		scaled, err := barcode.Scale(qrCode, 192, 192)
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		// QR images expose 16-bit grayscale colors; gofpdf accepts only
		// 8-bit PNGs. Convert explicitly before encoding.
		qrImage := image.NewGray(scaled.Bounds())
		draw.Draw(qrImage, qrImage.Bounds(), scaled, scaled.Bounds().Min, draw.Src)
		if err := png.Encode(&buf, qrImage); err != nil {
			return err
		}
		imgName := "qr-" + token.Code
		p.RegisterImageOptionsReader(imgName, gofpdf.ImageOptions{ImageType: "PNG"}, &buf)
		p.ImageOptions(imgName, x+2, y+2, 24, 24, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
		p.SetFont("Helvetica", "B", 12)
		p.SetXY(x+28, y+2)
		p.Cell(cellW-30, 6, "pixelbonus.com")
		p.SetFont("Courier", "", 10)
		midpoint := len(token.Code) / 2
		p.SetXY(x+28, y+11)
		p.Cell(cellW-30, 5, groupCode(token.Code[:midpoint]))
		p.SetXY(x+28, y+16)
		p.Cell(cellW-30, 5, groupCode(token.Code[midpoint:]))
	}
	return p.OutputFileAndClose(outputPath)
}

func groupCode(code string) string {
	var out []byte
	for i := 0; i < len(code); i += 4 {
		if i > 0 {
			out = append(out, ' ')
		}
		end := i + 4
		if end > len(code) {
			end = len(code)
		}
		out = append(out, code[i:end]...)
	}
	return string(out)
}
