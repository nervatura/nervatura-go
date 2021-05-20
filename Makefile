APP_NAME = nervatura
VNUM = 5.0.0-beta-1
# all http grpc postgres mysql sqlite
TAGS = all

security:
	gosec -quiet ./...

test: security
	go test -cover ./...

build:
	CGO_ENABLED=0 go build -tags "$(TAGS)" -ldflags="-w -s -X main.Version=$(VNUM)" -o $(APP_NAME) main.go

run:
	./$(APP_NAME)

clean:
	rm -rf ./${APP_NAME}

docs:
	godoc -http=:6060

docker.build:
	docker build -t nervatura --build-arg APP_MODULES=$(TAGS) .

docker.run:
	docker run -i -t --rm \
		--name nervatura \
		-p 5000:5000 \
		-v $(PWD)/data:/data \
		nervatura:latest

release: build
	upx --best --lzma nervatura

deploy:
	git add .
	git commit -m 'v${VNUM}'
	git push
