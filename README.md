# syncutils

[![license](https://img.shields.io/github/license/esome/golang-sync?style=flat&label=License&labelColor=rgb(45%2C%2049%2C%2054)&color=rgb(113%2C%2016%2C%20126))](LICENSE.md)
[![release](https://img.shields.io/github/v/release/esome/golang-sync?include_prereleases&sort=date&display_name=release&style=flat&label=Release&labelColor=rgb(45%2C%2049%2C%2054)&logo=GitHub&logoColor=rgb(136%2C%20142%2C%20147))](https://github.com/esome/golang-sync/releases)
[![badge](https://github.com/esome/golang-sync/workflows/CodeQL/badge.svg)](https://github.com/esome/golang-sync/actions/workflows/github-code-scanning/codeql)
[![badge](https://github.com/esome/golang-sync/workflows/Go/badge.svg)](https://github.com/esome/golang-sync/actions/workflows/go.yml)
![badge](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/sGy1980de/b272dbf4526c9be75f7da96352873a71/raw/golang-sync-coverage.json)

This package provides thread-safe data structures and other utilities for working with concurrency.

😎 Features:

* Type-safe, thanks to generics
* Zero-dependency

## 1.1. 📦 Installation

```bash
go get github.com/esome/golang-sync
```

## 1.2. 🛠️ Usage

### 1.2.1. `syncutils.Slice`

Slice is a thread-safe wrapper around a slice. It provides all methods needed to manipulate the slice in a thread-safe way.
For that matter it uses a `sync.Mutex` for locking purposes and uses the [Go 1.21 slices package][go121-slices] for changing the slice.

```go
import "github.com/esome/golang-sync"

var s1 Slice[string]           // thread-safe nil []string slice  
s2 := NewSlice[string](10)     // thread-safe variant of make([]string, 10)
s3 := NewSlice[string](0, 10)  // thread-safe variant of make([]string, 0, 10)

s1.Append("a", "b", "c")       // appends multiple elements
s1.Replace(1, 2, "d")          // replaces multiple elements - results in ["a", "d"]

for range s1.Iter() {          // iterates over the slice in a thread-safe non-blocking way
    // do something            // for that matter, the slice might be changed during the iteration, 
}                              // and the elements do not necessarily reflect any consistent state of the slice    
```

### 1.2.2. `syncutils.Map`

`Map` is a thread-safe wrapper around a map. It provides all methods needed to manipulate the map in a thread-safe way.
For that matter it uses a `sync.Mutex` for locking purposes.

```go
import "github.com/esome/golang-sync"

var m1 Map[string, string]        // thread-safe map[string]string  
m2 := NewMap[string, string](10)  // thread-safe variant of make(map[string]string, 10)

m1.Store("foo", "bar")                // same as m1["foo"] = "bar"
v, ok := m1.Load("foo")               // same as v, ok := m1["foo"]
v, ok := m1.LoadOrStore("foo", "baz") // read value of m1["foo"]. When not found store "baz" and return it 
v, ok := m1.LoadAndDelete("foo")      // removes the value and returns it

for range m1.Iter() {            // iterates over the map in a thread-safe non-blocking way
    // do something              // for that matter, the map might be changed during the iteration, 
}                                // and the elements do not necessarily reflect any consistent state of the map
```

### 1.2.3. `syncutils.MapCmp`

`MapCmp` extends the `Map` type with utility methods for deleting or swapping values only when the current value
matches an expected one. For this to be possible, the value type must be `comparable`.

```go
import "github.com/esome/golang-sync"

var m1 MapCmp[string, string]        // thread-safe map[string]string  

m1.Store("foo", "bar")              // same as m1["foo"] = "bar"
v, ok := m1.Load("foo")             // same as v, ok := m1["foo"]

m1.SwapIfEqual("foo", "baz", "bar") // swaps the value only if it is "bar"
m1.DeleteIfEqual("foo", "bar")      // deletes the value only if it is "bar"
```
### 1.2.4. `channels` package

The `channels` package provides a set of utilities for working with channels.
Please refer to the docs, to learn more about the available functions.  

```go
import (
    "context"
    
    "github.com/esome/golang-sync/channels"
)

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

ints := make(chan int)  
err := channels.Send(ctx, ints, 1, 2, 3, 4)  // sends values to the channel, returns an error if the context is done

recv1, recv2 := make(chan int), make(chan int)   
err := channels.Broadcast(ctx, ints, recv1, recv2)    // broadcasts each value from the ints channel to every recv* channel

// wraps a channel with a function that transforms the values
// uints is of type (<-chan uint) and contains the transformed values
uints := channels.Wrap(ctx, ints, func(_ context.Context, i int) ) (uint, bool) {
	return uint(i), i >= 0 // avoid overflow - if false, the value is skipped
})

```

[go121-slices]: https://tip.golang.org/doc/go1.21#slices
