package tests

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"pixelbonus/src/pixelbonus/coupon"
	"pixelbonus/src/pixelbonus/pdf"
)

func TestRenderCouponsCreatesPDF(t *testing.T) {
	output := filepath.Join(t.TempDir(), "coupons.pdf")
	tokens := []coupon.CouponToken{{Normalized: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", Grouped: "AAAA AAAA AAAA AAAA AAAA AAAA AAAA AAAA AAAA AAAA"}}
	if err := pdf.RenderCoupons(output, "Course", "https://pixelbonus.test", tokens); err != nil {
		t.Fatal(err)
	}
	data, err := ioutil.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(data[:4]) != "%PDF" {
		t.Fatalf("unexpected file header: %q", string(data[:4]))
	}
}
