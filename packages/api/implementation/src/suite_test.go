package main

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestAPI is the Ginkgo entry point for the API package test suites.
//
// Per specification/main.md#verification, the test suites are written using Go's built-in
// testing package (this function, driven by `go test`), Ginkgo (the spec tree
// declared via Describe/It across the *_test.go files) and Gomega (the
// matchers used in the specs). A single RunSpecs call collects and runs every
// spec registered in this package.
func TestAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Suite")
}
