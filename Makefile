serve:
	docker run -it --rm \
		-v $(PWD):/gowaker \
		-v gopkgcache:/go/pkg/mod \
		--env-file $(PWD)/secrets.list \
		-w /gowaker \
		-p 8080:8080 \
		golang:1.13 \
		go run . --logtostderr

pi:
	docker run -it --rm \
		-v $(PWD):/gowaker \
		-w /gowaker \
		-e GOOS=linux \
		-e GOARCH=arm \
		-e GOARM=5 \
		golang:1.13 \
		go build -o gowaker

deploy:
	scp gowaker waker:
