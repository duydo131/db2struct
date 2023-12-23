GO_CMD_MAIN = .

# If the first argument is "db2struct"...
#ifeq (db2struct, $(firstword $(MAKECMDGOALS)))
#  # use the rest as arguments for "run"
#  RUN_ARGS := $(wordlist 2, $(words $(MAKECMDGOALS)), $(MAKECMDGOALS))
#  # ...and turn them into do-nothing targets
#  @echo "Number of words: $(RUN_ARGS)"
#  $(eval $(RUN_ARGS):;@:)
#endif

db2struct:
	go run $(GO_CMD_MAIN) $(MAKEFLAGS)

all:
    @echo "Make version: $(MAKE_VERSION)"
	@echo "Makeflags: $(MAKEFLAGS)"
	@echo "Environment variables:"
	@env | grep '^MAKE'