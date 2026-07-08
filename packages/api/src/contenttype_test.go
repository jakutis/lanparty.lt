package main

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("contentTypeFor", func() {
	DescribeTable("maps supported types to a content type and extension",
		func(typ, wantCT, wantExt string) {
			ct, ext, ok := contentTypeFor(typ)
			Expect(ok).To(BeTrue())
			Expect(ct).To(Equal(wantCT))
			Expect(ext).To(Equal(wantExt))
		},
		Entry("html", "html", "text/html; charset=utf-8", ".html"),
		Entry("is case-insensitive", "HTML", "text/html; charset=utf-8", ".html"),
		Entry("markdown", "markdown", "text/markdown; charset=utf-8", ".md"),
	)

	DescribeTable("reports every other type as unsupported",
		func(typ string) {
			_, _, ok := contentTypeFor(typ)
			Expect(ok).To(BeFalse())
		},
		Entry("htm", "htm"),
		Entry("md", "md"),
		Entry("pdf", "pdf"),
		Entry("json", "json"),
		Entry("unknown", "totally-unknown-type"),
	)
})
