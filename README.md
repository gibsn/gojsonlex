# GoJSONLex

`gojsonlex` is a drop in replacement for `encoding/json` lexer (with `SkipDelims` set to `true`)
optimised for efficiency. `gojsonlex` is 2-3 times faster than `encoding/json` and requires memory
only enough to buffer the longest token in the input.

# API Documentation

https://pkg.go.dev/github.com/gibsn/gojsonlex

# Motivation

Let's consider a case when you want to parse the output of some tool that encodes binary data to one huge JSON dict:
```
{
  "bands": [
    {
      "name": "Metallica",
      "origin": "USA",
      "albums": [
        ...
      ]
    },
    ...
    {
      "name": "Enter Shikari",
      "origin": "England"
      "albums": [
        ...
      ]
    }
  ]
}
```

Let's say "albums" can be arbitrary long, the whole JSON is 10GB, but you actually want to print out all "origin" values and don't care about the rest. You do not want to decode the whole JSON into one struct (like most JSON parsers do) since it can be huge. Luckily in this case you do not actually need to parse any arbitrary JSON, you are ok with a more narrow grammar. A parser for such a grammar could look like this:

```golang
for {
	currToken, err := lexer.Token()
	if err != nil {
		// ...
	}

	switch state {
	case searchingForOriginKey:
		if currToken == "origin" {
			state := pendingOriginValue
		}
	case pendingOriginValue:
		fmt.Println(currToken)
		state = searchingForOriginKey
	}
}
```

Ok, so now you need a JSON lexer. Some lexers that I checked did buffer a large portion of input in order to parse a composite type (which is bad since "albums" can be huge). The only lexer that did not require that much memory was the standard `encoding/json`, however it could be optimized to consume less CPU. That's how `gojsonlex` was born.

# Overview

Example from the previous section could be implemented with `gojsonlex` like this:
```golang
l, err := gojsonlex.NewJSONLexer(r)
if err != nil {
	// ...
}

state = stateSearchingForOriginKey

for {
	currToken, err := lexer.Token()
	if err != nil {
		// ...
	}
	
	s, ok := currToken.(string)
	if !ok {
		continue
	}

	switch state {
	case stateSearchingForOriginKey:
		if s == "origin" {
			state := pendingOriginValue
		}
	case statePendingOriginValue:
		fmt.Println(s)
		state = searchingForOriginKey
	}
}
```

In order to maintain zero allocations `Token()` will always return an unsafe string that is valid only until the next `Token()` call. You must make a deep copy (using `StringDeepCopy()`) of that string in case you may need it after the next `Token()` call.

Though `gojsonlex.Token()` is faster than that from `encoding/json`, it sacfrifices performance in order to match the default interface. You may want to consider using `TokenFast()` to achieve the best performance (in exchange for more coding):
```golang
for {
	currToken, err := lexer.TokenFast()
	if err != nil {
		// ...
	}
	
	if currToken.Type() != LexerTokenTypeString {
		continue
	}
	
	s := currToken.String()

	switch state {
	case stateSearchingForOriginKey:
		if s == "origin" {
			state := pendingOriginValue
		}
	case statePendingOriginValue:
		fmt.Println(s)
		state = searchingForOriginKey
	}
}
```

# Examples
Please refer to the 'examples' directory for the examples of `gojsonlex` usage. Run `make examples` to build all examples.

## stdinparser
`stdinparser` is a simple utility that reads JSON from StdIn and dumps JSON tokens to StdOut


# Benchmarks
```
BenchmarkEncodingJSON-8    	     644	   1605048 ns/op	  342976 B/op	   21906 allocs/op
BenchmarkJSONLexer-8       	    1602	    737482 ns/op	   86400 B/op	    5500 allocs/op
BenchmarkJSONLexerFast-8   	    1963	    571005 ns/op	       0 B/op	       0 allocs/op
```

# Status

In development
