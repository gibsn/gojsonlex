`gojsonlex` is a fast drop in replacement for encoding/json's lexer. 

# Motivation

Let's consider a case when you want to parse the output of some tool that encodes binary data to one huge JSON dict:
```
{
  "bands": [
    {
      "name": "Metallica",
      "origin": "USA",
      "albums": [
        // all albums here
      ]
    },
    // ...
    {
      "name": "Enter Shikari",
      "origin": "England"
      "albums": [
        // all albums here
      ]
    }
  ]
}
```

Let's say "albums" can be arbitrary long, the whole JSON is 10GB, but you actually want to print out all "origin" values and don't care about the rest. All JSON parsers that I checked are subject to 2 main problems:
* library API requires the whole input in memory (which is bad since our JSON is huge);
* a large portion of input is bufferised in order to parse a composite type (which is bad since "albums" can be huge).

In this concrete case you do not actually need to parse any arbitrary JSON, you are ok with a more narrow grammar. A parser for such a grammar could look like this:

```
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

Ok, so now we need some JSON lexer that will be free of the problems described. Actually, the standard `encoding/json` package provides such a lexer, but it could be optimized to consume less CPU. That's how `gojsonlex` was born.

# Overview

TODO


# Benchmarks
```
BenchmarkEncodingJSON-8    	     644	   1605048 ns/op	  342976 B/op	   21906 allocs/op
BenchmarkJSONLexer-8       	    1602	    737482 ns/op	   86400 B/op	    5500 allocs/op
BenchmarkJSONLexerFast-8   	    1963	    571005 ns/op	       0 B/op	       0 allocs/op
```

# Status

In development
