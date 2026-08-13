CMD_DIR=cmd
BIN_DIR=bin
PUB_DIR=static/public

USER_SRC=$(CMD_DIR)/user_api/main.go
USER_BIN=$(BIN_DIR)/user

STAFF_SRC=$(CMD_DIR)/staff_api/main.go
STAFF_BIN=$(BIN_DIR)/staff

STYLE_SRC=$(PUB_DIR)/style.scss
STYLE_DST=$(PUB_DIR)/style.css

DATA_DB=$(BIN_DIR)/data.db

LAST_BUILD=.last_build

PORT=8080

.PHONY: run clean

all: style.css user staff

user:
	mkdir -p $(BIN_DIR)
	go build -o $(USER_BIN) $(USER_SRC)
	echo $(USER_BIN) > $(LAST_BUILD)

staff:
	mkdir -p $(BIN_DIR)
	go build -o $(STAFF_BIN) $(STAFF_SRC)
	echo $(STAFF_BIN) > $(LAST_BUILD)

style.css:
	sass $(STYLE_SRC) $(STYLE_DST)

run:
	./$$(cat $(LAST_BUILD) 2>/dev/null) -port $(PORT)

clean:
	rm -rf $(BIN_DIR)
	rm -rf $(STYLE_DST)
