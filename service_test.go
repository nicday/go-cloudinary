// Copyright 2013 Mathias Monnerville and Anthony Baillard.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package cloudinary

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDial(t *testing.T) {
	if _, err := Dial("baduri::"); err == nil {
		t.Error("should fail on bad uri")
	}

	// Not a cloudinary:// URL scheme
	if _, err := Dial("http://localhost"); err == nil {
		t.Error("should fail if URL scheme different from cloudinary://")
	}

	// Missing API secret (password)?
	if _, err := Dial("cloudinary://login@cloudname"); err == nil {
		t.Error("should fail when no API secret is provided")
	}

	k := &Service{
		cloudName: "cloudname",
		apiKey:    "login",
		apiSecret: "secret",
	}
	s, err := Dial(fmt.Sprintf("cloudinary://%s:%s@%s", k.apiKey, k.apiSecret, k.cloudName))
	if err != nil {
		t.Error("expect a working service at this stage but got an error.")
	}
	if s.cloudName != k.cloudName || s.apiKey != k.apiKey || s.apiSecret != k.apiSecret {
		t.Errorf("wrong service instance. Expect %v, got %v", k, s)
	}
	uexp := fmt.Sprintf("%s/%s/image/upload/", baseUploadUrl, s.cloudName)
	if s.uploadURI.String() != uexp {
		t.Errorf("wrong upload URI. Expect %s, got %s", uexp, s.uploadURI.String())
	}

}

func TestVerbose(t *testing.T) {
	s := new(Service)
	s.Verbose(true)
	if !s.verbose {
		t.Errorf("wrong verbose attribute. Expect %v, got %v", true, s.verbose)
	}
}

func TestSimulate(t *testing.T) {
	s := new(Service)
	s.Simulate(true)
	if !s.simulate {
		t.Errorf("wrong simulate attribute. Expect %v, got %v", true, s.simulate)
	}
}

func TestUploadURI(t *testing.T) {
	mockServerRequested := false
	server := mockCloudinaryServer(&mockServerRequested)
	defer server.Close()

	s := cloudinaryService()

	err := s.UploadURI(server.URL)
	if err != nil {
		t.Error("expected to set the upload URI but got an error")
	}

	u, err := s.UploadImage("test", strings.NewReader(""), "")
	if err != nil {
		t.Error("expected no error to occur", err)
	}

	if !mockServerRequested {
		t.Error("expected mock Cloudinary service to be requested")
	}

	expectedURL := "tests/test_file"
	if u != expectedURL {
		t.Errorf("expected returned url %s to match %s", u, expectedURL)
	}
}

func TestKeepFiles(t *testing.T) {
	s := new(Service)
	if err := s.KeepFiles(""); err != nil {
		t.Error("empty pattern should not raise an error")
	}
	pat := "[[;"
	if err := s.KeepFiles(pat); err == nil {
		t.Errorf("wrong pattern %s should raise an error", pat)
	}
	pat = "images/\\.jpg$"
	err := s.KeepFiles(pat)
	if err != nil {
		t.Errorf("valid pattern %s should return no error", pat)
	}
	if s.keepFilesPattern == nil {
		t.Errorf(".keepFilesPattern attribute is still nil with a valid pattern")
	}
}

func TestUseDatabase(t *testing.T) {
	s := new(Service)
	if err := s.UseDatabase("baduri::"); err == nil {
		t.Error("should fail on bad uri")
	}
	// Bad scheme
	if err := s.UseDatabase("http://localhost"); err == nil {
		t.Error("should fail if URL scheme different from mongodb://")
	}
	if err := s.UseDatabase("mongodb://localhost/cloudinary"); err != nil {
		t.Error("please ensure you have a running MongoDB server on localhost")
	}
	if s.dbSession == nil || s.col == nil {
		t.Error("service's dbSession and col should not be nil")
	}
}

func TestCleanAssetName(t *testing.T) {
	assets := [][4]string{
		// order: path, basepath, prepend, expected result
		{"/tmp/css/default.css", "/tmp/", "new", "new/css/default"},
		{"/a/b/c.png", "/a", "", "b/c"},
		{"/a/b/c.png", "/a ", "  ", "b/c"}, // With spaces
		{"/a/b/c.png", "", "/x", "x/a/b/c"},
	}
	for _, p := range assets {
		c := cleanAssetName(p[0], p[1], p[2])
		if c != p[3] {
			t.Errorf("wrong cleaned name. Expect '%s', got '%s'", p[3], c)
		}
	}
}

func TestPublicID(t *testing.T) {
	urls := [][2]string{
		// order: url, expected result
		{"http://res.cloudinary.com/cloud-name/image/upload/857477010", "857477010"},
		{"http://res.cloudinary.com/cloud-name/image/upload", ""},
		{"http://res.cloudinary.com/cloud-name/image/upload/", ""},
		{"http://res.cloudinary.com/cloud-name/image/upload/something/extra", ""},
	}

	s := &Service{
		cloudName: "cloudname",
		apiKey:    "login",
		apiSecret: "secret",
	}

	for _, u := range urls {
		id, err := s.PublicID(u[0])

		if u[1] == "" && err != ErrUnexpectedURLPathFormat {
			t.Errorf("wrong error returned. Expect '%s', got '%s'", u[1], err)
		}

		if id != u[1] {
			t.Errorf("wrong public ID. Expect '%s', got '%s'", u[1], id)
		}
	}
}

// mockCloudinaryServer is a server that always responds with a successful image upload respose.
func mockCloudinaryServer(called *bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")

		fmt.Fprintln(w, `{"public_id":"tests/test_file","version":1369431906,"format":"png","resource_type":"image"}`)
	}))
}

// cloudinaryService returns a basic Service
func cloudinaryService() *Service {
	k := &Service{
		cloudName: "cloudname",
		apiKey:    "login",
		apiSecret: "secret",
	}

	s, err := Dial(fmt.Sprintf("cloudinary://%s:%s@%s", k.apiKey, k.apiSecret, k.cloudName))
	if err != nil {
		log.Fatal("expect a working service at this stage but got an error.")
	}

	return s
}
