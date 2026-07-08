package main

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("contentTypeFor", func() {
	DescribeTable("maps known types to a content type and extension",
		func(typ, wantCT, wantExt string) {
			ct, ext := contentTypeFor(typ)
			Expect(ct).To(Equal(wantCT))
			Expect(ext).To(Equal(wantExt))
		},
		Entry("html", "html", "text/html; charset=utf-8", ".html"),
		Entry("is case-insensitive", "HTML", "text/html; charset=utf-8", ".html"),
		Entry("htm", "htm", "text/html; charset=utf-8", ".html"),
		Entry("json", "json", "application/json; charset=utf-8", ".json"),
		Entry("txt", "txt", "text/plain; charset=utf-8", ".txt"),
		Entry("md", "md", "text/markdown; charset=utf-8", ".md"),
		Entry("svg", "svg", "image/svg+xml", ".svg"),
	)

	It("falls back to application/octet-stream for unknown types", func() {
		ct, ext := contentTypeFor("totally-unknown-type")
		Expect(ct).To(Equal("application/octet-stream"))
		Expect(ext).To(Equal(".totally-unknown-type"))
	})
})
