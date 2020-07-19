# gojsonlex 

gojsonlex is a fast drop in replacement for encoding/json's lexer. 

# Motivation

gojsonlex can be mainly used to implement your own JSON parser. Let's consider a case when you want to parse the output of some tool that encodes binary data to one JSON dict:
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

Let's say you want to print out 
```


When working with JSON in most cases you will be satisfied with one of the many fast decoding/encoding libraries implemented, but lets say you want to parse 


# API




# Benchmarks
```
BenchmarkEncodingJSON-8    	     644	   1605048 ns/op	  342976 B/op	   21906 allocs/op
BenchmarkJSONLexer-8       	    1602	    737482 ns/op	   86400 B/op	    5500 allocs/op
BenchmarkJSONLexerFast-8   	    1963	    571005 ns/op	       0 B/op	       0 allocs/op
```

# Status

In development
