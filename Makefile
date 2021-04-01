APP_NAME = nervatura

security:
	gosec -quiet ./...

test: security
	go test -cover ./...

build:
	CGO_ENABLED=0 go build -tags "all" -ldflags="-w -s" -o $(APP_NAME) main.go

run:
	./$(APP_NAME)

clean:
	rm -rf ./${APP_NAME}

docs:
	godoc -http=:6060

docker.build:
	docker build -t nervatura --build-arg APP_MODULES=sqlite,postgres .

docker.run:
	docker run -i -t --rm \
		--name nervatura \
		-p 5000:5000 \
		-v $(PWD)/data:/data \
		nervatura:latest