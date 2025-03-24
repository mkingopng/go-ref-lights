//go:build remote
// +build remote

// file: test/referee_flow_remote_test.go
//
package test

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func closeRespBodyRemote(t *testing.T, body io.ReadCloser, context string) {
	if err := body.Close(); err != nil {
		t.Logf("[Remote] Warning: error closing response body for %s: %v", context, err)
	}
}

// TestRefereeFlowRemote calls the same flow but on a deployed environment.
// We expect environment variable REMOTE_BASE_URL to be set, e.g. "https://referee-lights.michaelkingston.com.au".
func TestRefereeFlowRemote(t *testing.T) {
	baseURL := os.Getenv("REMOTE_BASE_URL")
	if baseURL == "" {
		t.Skipf("[Remote] Skipping TestRefereeFlowRemote: REMOTE_BASE_URL not set.")
		return
	}
	t.Logf("[Remote] Using baseURL=%s", baseURL)

	// create an HTTP client with cookie jar for session
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("[Remote] Failed to create cookie jar: %v", err)
	}
	client := &http.Client{Jar: jar}

	// 1. GET /
	respA, errA := client.Get(baseURL + "/")
	if errA != nil {
		t.Fatalf("[Remote] GET / failed: %v", errA)
	}
	closeRespBodyRemote(t, respA.Body, "GET /")
	if respA.StatusCode != http.StatusOK {
		t.Fatalf("[Remote] GET / -> %d, want 200", respA.StatusCode)
	}

	// 2. POST /set-meet
	formData := url.Values{}
	formData.Set("meetName", "Dragon Cup 2") // todo: change to actual meet name
	respB, errB := client.PostForm(baseURL+"/set-meet", formData)
	if errB != nil {
		t.Fatalf("[Remote] POST /set-meet failed: %v", errB)
	}
	closeRespBodyRemote(t, respB.Body, "POST /set-meet")
	if respB.StatusCode != http.StatusFound && respB.StatusCode != http.StatusOK {
		t.Fatalf("[Remote] POST /set-meet -> %d, want 302 or 200", respB.StatusCode)
	}

	// 3. POST /login
	loginForm := url.Values{}
	loginForm.Set("username", "dragon_cup") // todo: change to actual username
	loginForm.Set("password", "YqW8qd")     // todo: change to actual password
	respC, errC := client.PostForm(baseURL+"/login", loginForm)
	if errC != nil {
		t.Fatalf("[Remote] POST /login failed: %v", errC)
	}
	closeRespBodyRemote(t, respC.Body, "POST /login")
	if respC.StatusCode != http.StatusFound && respC.StatusCode != http.StatusOK {
		t.Fatalf("[Remote] POST /login -> %d, want 302 or 200", respC.StatusCode)
	}

	// 4. GET /index
	respD, errD := client.Get(baseURL + "/index")
	if errD != nil {
		t.Fatalf("[Remote] GET /index failed: %v", errD)
	}
	defer closeRespBodyRemote(t, respD.Body, "GET /index")
	if respD.StatusCode != http.StatusOK {
		t.Fatalf("[Remote] GET /index -> %d, want 200", respD.StatusCode)
	}

	docD, errDocD := goquery.NewDocumentFromReader(respD.Body)
	if errDocD != nil {
		t.Fatalf("[Remote] parse /index: %v", errDocD)
	}
	qrCount := 0
	docD.Find(".qr-code-item").Each(func(i int, s *goquery.Selection) {
		qrCount++
	})
	if qrCount != 3 {
		t.Errorf("[Remote] Wanted 3 .qr-code-item, found %d", qrCount)
	}

	// 5. GET /referee/.../center
	refPath := "/referee/Dragon%20Cup%202/center" // todo
	respE, errE := client.Get(baseURL + refPath)
	if errE != nil {
		t.Fatalf("[Remote] GET referee page failed: %v", errE)
	}
	defer closeRespBodyRemote(t, respE.Body, "GET /referee/center (Remote)")
	if respE.StatusCode != http.StatusOK {
		t.Fatalf("[Remote] GET referee page -> %d, want 200", respE.StatusCode)
	}

	docE, errDocE := goquery.NewDocumentFromReader(respE.Body)
	if errDocE != nil {
		t.Fatalf("[Remote] parse referee page: %v", errDocE)
	}

	// check assets
	var assets []string
	docE.Find("link[rel='stylesheet']").Each(func(i int, s *goquery.Selection) {
		if href, ok := s.Attr("href"); ok {
			assets = append(assets, href)
		}
	})
	docE.Find("script[src]").Each(func(i int, s *goquery.Selection) {
		if src, ok := s.Attr("src"); ok {
			assets = append(assets, src)
		}
	})

	t.Logf("[Remote] Found %d assets on referee page", len(assets))

	for _, asset := range assets {
		assetURL := asset
		if strings.HasPrefix(asset, "/") {
			assetURL = baseURL + asset
		}
		respAsset, errAsset := client.Get(assetURL)
		if errAsset != nil {
			t.Errorf("[Remote] GET asset %q failed: %v", assetURL, errAsset)
			continue
		}
		closeRespBodyRemote(t, respAsset.Body, assetURL)
		if respAsset.StatusCode != http.StatusOK {
			t.Errorf("[Remote] Asset %q -> %d, want 200", assetURL, respAsset.StatusCode)
		}
	}
	t.Log("[Remote] TestRefereeFlowRemote complete!")
}
