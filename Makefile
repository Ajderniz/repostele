BIN_DIR=bin
PUB_DIR=static/public

MAIN_SRC=cmd/main.go

USER_BIN=$(BIN_DIR)/user
STAFF_BIN=$(BIN_DIR)/staff

STYLE_SRC=scss/style.scss
STYLE_DST=$(PUB_DIR)/css/style.css

DATA_DB=$(BIN_DIR)/data.db

LAST_BUILD=.last_build

# Windows build: a folder (and zip) to copy to the target machine.
# WIN_ARCH=386 for 32-bit Windows.
WIN_ARCH=amd64
WIN_NAME=repostele-windows
WIN_DIR=dist/$(WIN_NAME)
WIN_GO=GOOS=windows GOARCH=$(WIN_ARCH) CGO_ENABLED=0 go build

PORT=8080

.PHONY: all user staff style.css vet test windows run clean

all: style.css user staff

user:
	mkdir -p $(BIN_DIR)
	go build -tags=USER -o $(USER_BIN) $(MAIN_SRC)
	echo $(USER_BIN) > $(LAST_BUILD)

staff:
	mkdir -p $(BIN_DIR)
	go build -tags=STAFF -o $(STAFF_BIN) $(MAIN_SRC)
	echo $(STAFF_BIN) > $(LAST_BUILD)

style.css:
	sass --embed-sources $(STYLE_SRC) $(STYLE_DST)

vet:
	go vet -tags=USER ./...
	go vet -tags=STAFF ./...

# End-to-end: builds both apps into a temp dir with a fresh DB.
test: style.css
	go test -count=1 ./test/e2e

# .bat files get CRLF line endings; cmd.exe misreads labels without them.
windows: style.css
	rm -rf $(WIN_DIR) $(WIN_DIR).zip
	mkdir -p $(WIN_DIR)
	$(WIN_GO) -tags=USER -o $(WIN_DIR)/user.exe $(MAIN_SRC)
	$(WIN_GO) -tags=STAFF -o $(WIN_DIR)/staff.exe $(MAIN_SRC)
	for f in scripts/windows/*.bat; do \
	  sed 's/$$/\r/' $$f > $(WIN_DIR)/$$(basename $$f); \
	done

run:
	./$$(cat $(LAST_BUILD) 2>/dev/null) -port $(PORT)

clean:
	rm -f $(USER_BIN) $(STAFF_BIN)
	rm -f $(STYLE_DST)
