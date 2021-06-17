APP_NAME = nervatura
VNUM = 5.0.0-beta.2
# all http grpc postgres mysql sqlite
TAGS = all

security:
	gosec -quiet ./...

test: security
	go test -cover ./...

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags "$(TAGS)" -ldflags="-w -s -X main.version=$(VNUM)" -o $(APP_NAME) main.go

build-win:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags "$(TAGS)" -ldflags="-w -s -X main.version=$(VNUM)" -o $(APP_NAME).exe main.go

run:
	./$(APP_NAME)

clean:
	rm -rf ./${APP_NAME}

clean.test:
	go clean -testcache

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
	upx --best --lzma $(APP_NAME)

gorelease:
	GORELEASER_CURRENT_TAG=$(VNUM) goreleaser --snapshot --skip-publish --rm-dist

deploy:
	git add .
	git commit -m 'v${VNUM}'
	git push

snap-i:
	sudo snap install dist/nervatura_$(VNUM)_amd64.snap --dangerous

snap-u:
	sudo snap remove nervatura

snap-info:
	sudo systemctl status -l snap.nervatura.nervatura.service

snap-stop:
	sudo systemctl stop snap.nervatura.nervatura.service

snap-start:
	sudo systemctl start snap.nervatura.nervatura.service

snap-boot:
	journalctl -u snap.nervatura.nervatura.service -b

snap-demo:
	sudo NT_API_KEY=DEMO_API_KEY NT_ALIAS_DEMO="sqlite://file:/var/snap/nervatura/common/demo.db?cache=shared&mode=rwc" /snap/nervatura/current/nervatura -c DatabaseCreate -k DEMO_API_KEY -o "{\"database\":\"demo\",\"demo\":true}"
