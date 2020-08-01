EXAMPLES_SRCS=$(wildcard examples/*)
EXAMPLES_BINS=$(addprefix bin/, $(notdir $(basename $(EXAMPLES_SRCS))))

examples: $(EXAMPLES_BINS)

$(EXAMPLES_BINS):
	go build -o $@ github.com/gibsn/gojsonlex/examples/$(notdir $@)

test:
	go test .

bench:
	go test -bench=. -benchmem

clean:
	rm -rf ./bin

.PHONY: test bench examples clean
