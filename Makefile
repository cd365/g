.PHONY: code
code: betteralign fmt gofumpt

.PHONY: betteralign
betteralign:
	betteralign -apply -fix ./...
	betteralign -apply -fix ./es/...

.PHONY: gofumpt
gofumpt:
	gofumpt -w .
	gofumpt -w ./es/

.PHONY: fmt
fmt:
	for file in $$(find . -name "*.go"); do go fmt "$${file}"; done

.PHONY: install
install:
	go install github.com/dkorunic/betteralign/cmd/betteralign@latest
	go install mvdan.cc/gofumpt@latest
	go install github.com/axw/gocov/gocov@latest
	go install github.com/matm/gocov-html/cmd/gocov-html@latest