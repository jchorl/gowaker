serve:
	docker run -it --rm \
		-v $(PWD):/gowaker \
		-v gopkgcache:/go/pkg/mod \
		--env-file $(PWD)/secrets.list \
		-w /gowaker \
		golang:1.13 \
		go run main.go
