BIN_DIR=bin

USER_SRC_DIR=cmd/user_api
USER_SRC_FILENAME=main.go
USER_SRC=$(USER_SRC_DIR)/$(USER_SRC_FILENAME)

USER_BIN_FILENAME=user
USER_BIN=$(BIN_DIR)/$(USER_BIN_FILENAME)

STAFF_SRC_DIR=cmd/staff_api
STAFF_SRC_FILENAME=main.go
STAFF_SRC=$(STAFF_SRC_DIR)/$(STAFF_SRC_FILENAME)

STAFF_BIN_FILENAME=staff
STAFF_BIN=$(BIN_DIR)/$(STAFF_BIN_FILENAME)

DATA_DB=$(BIN_DIR)/data.db

LAST_BUILD=.last_build

PORT=8080

.PHONY: run clean

all: user staff

user:
	mkdir -p $(BIN_DIR)
	go build -o $(USER_BIN) $(USER_SRC)
	echo $(USER_BIN) > $(LAST_BUILD)

staff:
	mkdir -p $(BIN_DIR)
	go build -o $(STAFF_BIN) $(STAFF_SRC)
	echo $(STAFF_BIN) > $(LAST_BUILD)

run:
	./$$(cat $(LAST_BUILD) 2>/dev/null) -port $(PORT)

clean:
	rm -f $(USER_BIN) $(STAFF_BIN) $(DATA_DB)
