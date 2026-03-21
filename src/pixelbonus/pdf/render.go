package pdf

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"pixelbonus/src/pixelbonus/coupon"
)

func RenderCoupons(outputPath string, courseName string, baseURL string, tokens []coupon.CouponToken) error {
	if err := ensureDir(filepath.Dir(outputPath)); err != nil {
		return err
	}
	content := "%PDF-1.4\n"
	content += "1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n"
	content += "2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj\n"
	stream := "BT /F1 12 Tf 50 800 Td (" + escape(courseName) + ") Tj ET\n"
	stream += "BT /F1 10 Tf 50 780 Td (" + escape(baseURL) + ") Tj ET\n"
	y := 760
	for _, token := range tokens {
		stream += fmt.Sprintf("BT /F1 8 Tf 50 %d Td (%s) Tj ET\n", y, escape(token.Grouped))
		y -= 12
		if y < 40 {
			break
		}
	}
	content += fmt.Sprintf("3 0 obj << /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >> endobj\n")
	content += fmt.Sprintf("4 0 obj << /Length %d >> stream\n%sendstream endobj\n", len(stream), stream)
	content += "5 0 obj << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >> endobj\nxref\n0 6\n0000000000 65535 f \n"
	content += "trailer << /Root 1 0 R /Size 6 >>\nstartxref\n0\n%%EOF\n"
	return ioutil.WriteFile(outputPath, []byte(content), 0644)
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func escape(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "(", "\\(")
	value = strings.ReplaceAll(value, ")", "\\)")
	return value
}
