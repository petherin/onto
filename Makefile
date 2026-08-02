.PHONY: start

start:
	go run ./cmd/onto

.PHONY: regen-toc
regen-toc:
	# Regenerate README table-of-contents in-place using markdown-toc (npm)
	# Uses official Node image; requires network to fetch package via npx.
	docker run --rm -v $(PWD):/work -w /work node:18 npx -y markdown-toc -i README.md
