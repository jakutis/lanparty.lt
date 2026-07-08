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
		Entry("pdf", "pdf", "application/pdf", ".pdf"),
	)

	DescribeTable("reports every other type as unsupported",
		func(typ string) {
			_, _, ok := contentTypeFor(typ)
			Expect(ok).To(BeFalse())
		},
		Entry("htm", "htm"),
		Entry("json", "json"),
		Entry("txt", "txt"),
		Entry("unknown", "totally-unknown-type"),
	)
})
