setup: 
	sudo mkdir -p /etc/nvd/
	sudo mkdir -p /nmounts
	sudo cp nvd.json /etc/nvd/
	go get -u -v github.com/Nexenta/nexenta-docker-driver/...

lint:
	go get -v github.com/golang/lint/golint
	for file in $$(find $GOPATH/src/github.com/nexenta/nexenta-docker-driver -name '*.go' | grep -v vendor | grep -v '\.pb\.go' | grep -v '\.pb\.gw\.go'); do \
		golint $${file}; \
		if [ -n "$$(golint $${file})" ]; then \
			exit 1; \
		fi; \
	done

clean:
	go clean github.com/nexenta/nexenta-docker-driver
