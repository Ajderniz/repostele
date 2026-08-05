WEB_DIR=web

USER_SRC_DIR=cmd/user_api
USER_SRC_FILENAME=main.go
USER_SRC=$(USER_SRC_DIR)/$(USER_SRC_FILENAME)

USER_BIN_FILENAME=user
USER_BIN=$(WEB_DIR)/$(USER_BIN_FILENAME)

STAFF_SRC_DIR=cmd/staff_api
STAFF_SRC_FILENAME=main.go
STAFF_SRC=$(STAFF_SRC_DIR)/$(STAFF_SRC_FILENAME)

STAFF_BIN_DIR=web
STAFF_BIN_FILENAME=staff
STAFF_BIN=$(WEB_DIR)/$(STAFF_BIN_FILENAME)

DATA_DB=$(WEB_DIR)/assets/data.db

LAST_BUILD=.last_build

PORT=8080

.PHONY: user staff run clean

user:
	go build -o $(USER_BIN) $(USER_SRC)
	echo $(USER_BIN) > $(LAST_BUILD)

staff:
	go build -o $(STAFF_BIN) $(STAFF_SRC)
	echo $(STAFF_BIN) > $(LAST_BUILD)

run:
	./$$(cat $(LAST_BUILD) 2>/dev/null) -port $(PORT)

clean:
ifneq (,$(wildcard $(USER_BIN)))
	rm $(USER_BIN)
endif
ifneq (,$(wildcard $(STAFF_BIN)))
	rm $(STAFF_BIN)
endif
ifneq (,$(wildcard $(DATA_DB)))
	rm $(DATA_DB)
endif
