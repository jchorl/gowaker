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
	go build -o gowaker

deploy:
	rsync -avz --delete --exclude gowaker --exclude waker.db . waker:gowaker
