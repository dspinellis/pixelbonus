package tests

import (
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"pixelbonus/src/courses"
	"pixelbonus/src/tokens"
	"regexp"
	"strings"
	"testing"
	"time"

	"pixelbonus/src/pdf"
	"pixelbonus/src/redemption"
)

func TestCouponPDFWithRealQRImages(t *testing.T) {
	code := "00112233445566778899AABBCCDDEEFF00112233"
	coupons := make([]pdf.CouponPrintData, 20)
	for i := range coupons {
		coupons[i] = pdf.CouponPrintData{Code: code, URL: "https://www.pixelbonus.com:8443/r?c=" + redemption.EncodeTokenForURL(code)}
	}
	path := filepath.Join(t.TempDir(), "coupons.pdf")
	if err := (&pdf.Service{}).GenerateCoupons(testContext(), path, "", coupons); err != nil {
		t.Fatal(err)
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "%PDF-") || !strings.Contains(string(data), "/Subtype /Image") {
		t.Fatal("expected PDF containing QR images")
	}
}

type localCouponPDF struct{ path string }

func (p localCouponPDF) GenerateCoupons(ctx context.Context, _ string, prefix string, coupons []pdf.CouponPrintData) error {
	return (&pdf.Service{}).GenerateCoupons(ctx, p.path, prefix, coupons)
}

type couponNotification struct{ messages chan string }

func (m couponNotification) Send(_ context.Context, _ string, _ string, body string) error {
	m.messages <- body
	return nil
}

func TestCouponGenerationNotifiesInstructor(t *testing.T) {
	db := newTestDB(t)
	id := upsertTestInstructor(t, db)
	courseSvc := &courses.Service{DB: db}
	course, err := courseSvc.CreateCourse(testContext(), id, courses.CreateCourseInput{Name: "PDF coupons", GradingMethod: courses.ValidGradingMethods[0]})
	if err != nil {
		t.Fatal(err)
	}
	mailer := couponNotification{messages: make(chan string, 1)}
	svc := &tokens.Service{DB: db, Courses: courseSvc, Mailer: mailer, PDF: localCouponPDF{path: filepath.Join(t.TempDir(), "coupons.pdf")}, BaseURL: "https://example.com"}
	set, err := svc.CreateTokenSet(testContext(), id, course.ID, 20, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	select {
	case body := <-mailer.messages:
		if body != "https://example.com/download/"+set.FileNonce {
			t.Fatalf("wrong notification: %s", body)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("coupon generation never notified instructor")
	}
	if _, _, err := svc.GetDownloadPath(testContext(), set.FileNonce); err != nil {
		t.Fatalf("notified before coupons were ready: %v", err)
	}
}

// Exercise both the bottom row and the transition to the next sheet with
// distinct codes so every coupon can be checked for placement and splitting.
func TestCouponSheetPaginationAndText(t *testing.T) {
	for _, count := range []int{20, 40} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			coupons := make([]pdf.CouponPrintData, count)
			for i := range coupons {
				code := fmt.Sprintf("%020X%020X", i+1, i+1001)
				coupons[i] = pdf.CouponPrintData{Code: code, URL: "https://example.com/r?c=" + redemption.EncodeTokenForURL(code)}
			}
			path := filepath.Join(t.TempDir(), "coupons.pdf")
			if err := (&pdf.Service{}).GenerateCoupons(testContext(), path, "", coupons); err != nil {
				t.Fatal(err)
			}
			data, err := ioutil.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			pages := regexp.MustCompile(`/Type /Page\b`).FindAll(data, -1)
			if len(pages) != count/20 {
				t.Fatalf("%d coupons produced %d pages", count, len(pages))
			}
			var content []string
			for _, match := range regexp.MustCompile(`(?s)stream\n(.*?)\nendstream`).FindAllSubmatch(data, -1) {
				reader, err := zlib.NewReader(bytes.NewReader(match[1]))
				if err != nil {
					continue
				}
				stream, err := ioutil.ReadAll(reader)
				reader.Close()
				if err != nil {
					t.Fatal(err)
				}
				if bytes.Contains(stream, []byte("(pixelbonus.com)")) {
					content = append(content, string(stream))
				}
			}
			if len(content) != count/20 {
				t.Fatalf("got %d coupon content streams", len(content))
			}
			for page, stream := range content {
				if strings.Count(stream, "(pixelbonus.com)") != 20 {
					t.Fatalf("page %d does not contain all 20 labels", page+1)
				}
				for _, coupon := range coupons[page*20 : (page+1)*20] {
					for _, half := range []string{coupon.Code[:20], coupon.Code[20:]} {
						var groups []string
						for pos := 0; pos < 20; pos += 4 {
							groups = append(groups, half[pos:pos+4])
						}
						if !strings.Contains(stream, "("+strings.Join(groups, " ")+")") {
							t.Fatalf("page %d missing code half %s", page+1, half)
						}
					}
				}
			}
		})
	}
}
