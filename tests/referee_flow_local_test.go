//go:build integration
// +build integration

package test

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"go-ref-lights/internal/app"
)

// helper: closes a response body and logs any error
func closeRespBody(t *testing.T, body io.ReadCloser, context string) {
	if err := body.Close(); err != nil {
		t.Logf("Warning: error closing response body for %s: %v", context, err)
	}
}

// TestRefereeFlow simulates the real user flow of:
//	A) GET / -> choose-meet
//	B) POST /set-meet
//	C) POST /login
//	D) GET /index -> verifying there's 3 QR code items
//	E) GET /referee/.../center -> verifying the page loads CSS/JS properly

func TestRefereeFlow(t *testing.T) {
	// 1. spin up your Gin router in test mode
	t.Log("[TestRefereeFlow] Starting local test server in 'test' mode")
	router := app.SetupRouter("test") // pass "test" or "development"
	server := httptest.NewServer(router)
	defer server.Close()

	// 2. create an HTTP client with a cookie jar (to store session)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("Failed to create cookie jar: %v", err)
	}
	client := &http.Client{Jar: jar}

	baseURL := server.URL // todo: change to the actual base URL
	t.Logf("[TestRefereeFlow] Base URL is %s", baseURL)

	// A) GET / (choose-meet)
	t.Log("[TestRefereeFlow] Step A: GET /")
	respA, errA := client.Get(baseURL + "/")
	if errA != nil {
		t.Fatalf("GET / failed: %v", errA)
	}
	closeRespBody(t, respA.Body, "Step A (GET /)")

	if respA.StatusCode != http.StatusOK {
		t.Fatalf("GET / returned %d, want 200", respA.StatusCode)
	}

	// B) POST /set-meet -> "Dragon Cup 2"
	t.Log("[TestRefereeFlow] Step B: POST /set-meet for 'Dragon Cup 2'")
	formData := url.Values{}
	formData.Set("meetName", "Dragon Cup 2") // or an actual test meet name
	respB, errB := client.PostForm(baseURL+"/set-meet", formData)
	if errB != nil {
		t.Fatalf("POST /set-meet failed: %v", errB)
	}
	closeRespBody(t, respB.Body, "Step B (POST /set-meet)")

	// expect redirect (302) or 200 depending on your code
	if respB.StatusCode != http.StatusFound && respB.StatusCode != http.StatusOK {
		t.Fatalf("POST /set-meet returned %d (expected 302 or 200)", respB.StatusCode)
	}

	// C) POST /login with meet director’s username/password
	t.Log("[TestRefereeFlow] Step C: POST /login with credentials")
	loginForm := url.Values{}
	loginForm.Set("username", "dragon_cup") // todo: actual admin user
	loginForm.Set("password", "YqW8qd")     // todo: actual password

	respC, errC := client.PostForm(baseURL+"/login", loginForm)
	if errC != nil {
		t.Fatalf("POST /login failed: %v", errC)
	}
	closeRespBody(t, respC.Body, "Step C (POST /login)")

	// usually we expect a redirect to /index if success
	if respC.StatusCode != http.StatusFound && respC.StatusCode != http.StatusOK {
		t.Fatalf("POST /login returned %d, want 302 or 200", respC.StatusCode)
	}

	// D) GET /index -> confirm there are 3 QR code items
	t.Log("[TestRefereeFlow] Step D: GET /index")
	respD, errD := client.Get(baseURL + "/index")
	if errD != nil {
		t.Fatalf("GET /index failed: %v", errD)
	}
	defer closeRespBody(t, respD.Body, "Step D (GET /index)")

	if respD.StatusCode != http.StatusOK {
		t.Fatalf("GET /index returned %d, want 200", respD.StatusCode)
	}

	// parse /index to confirm there's something about the 3 positions or QR codes
	t.Log("[TestRefereeFlow] Parsing /index HTML to confirm presence of QR code items")
	docD, errDocD := goquery.NewDocumentFromReader(respD.Body)
	if errDocD != nil {
		t.Fatalf("Failed to parse /index page HTML: %v", errDocD)
	}

	// count how many .qr-code-item we find
	qrCount := 0
	docD.Find(".qr-code-item").Each(func(i int, s *goquery.Selection) {
		qrCount++
	})
	if qrCount != 3 {
		t.Errorf("[TestRefereeFlow] Expected 3 .qr-code-item elements on /index, found %d", qrCount)
	} else {
		t.Logf("[TestRefereeFlow] Found %d .qr-code-item elements on /index as expected", qrCount)
	}

	// E) GET /referee/Dragon%20Cup%202/center
	t.Log("[TestRefereeFlow] Step E: GET /referee/Dragon%20Cup%202/center")
	refereePath := "/referee/Dragon%20Cup%202/center"
	respE, errE := client.Get(baseURL + refereePath)
	if errE != nil {
		t.Fatalf("GET referee page failed: %v", errE)
	}
	defer closeRespBody(t, respE.Body, "Step E (GET /referee/...)")

	if respE.StatusCode != http.StatusOK {
		t.Fatalf("Referee page returned %d, want 200", respE.StatusCode)
	}

	// parse the HTML
	doc, err := goquery.NewDocumentFromReader(respE.Body)
	if err != nil {
		t.Fatalf("Failed to parse referee page HTML: %v", err)
	}

	// grab CSS/JS references
	var assets []string
	doc.Find("link[rel='stylesheet']").Each(func(i int, s *goquery.Selection) {
		if href, ok := s.Attr("href"); ok {
			assets = append(assets, href)
		}
	})
	doc.Find("script[src]").Each(func(i int, s *goquery.Selection) {
		if src, ok := s.Attr("src"); ok {
			assets = append(assets, src)
		}
	})

	t.Logf("[TestRefereeFlow] Found %d static assets to check", len(assets))

	// check each asset returns 200
	for _, asset := range assets {
		assetURL := asset
		if strings.HasPrefix(asset, "/") {
			assetURL = baseURL + asset
		}
		t.Logf("[TestRefereeFlow] Checking asset URL: %s", assetURL)

		respAsset, errAsset := client.Get(assetURL)
		if errAsset != nil {
			t.Errorf("[TestRefereeFlow] Failed to load asset %q: %v", assetURL, errAsset)
			continue
		}
		closeRespBody(t, respAsset.Body, "Asset: "+assetURL)

		if respAsset.StatusCode != http.StatusOK {
			t.Errorf("[TestRefereeFlow] Asset %q returned %d, want 200", assetURL, respAsset.StatusCode)
		}
	}
	t.Log("[TestRefereeFlow] Finished successfully!")
}
