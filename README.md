# epub

Golang epub reader library

## Summary

- Written on pure Go
- Require Go version >= 1.14

## Install

```plaintext
go get github.com/taofei-pro/epub-reader
```

## Usage

```golang
package main

import (
  "fmt"
  "os"

  "github.com/taofei-pro/epub-reader"
)

func main() {
  bk, err := Open("./data/test.epub")
  if err != nil {
    panic(err)
  }
  defer bk.Close()

  chapters := bk.NavPoints()
  for _, chapter := range chapters {
    fmt.Printf("title: %s\n", chapter.Text)

    content := bk.NavPointContent(chapter)
    fmt.Printf("content: %s\n", content)
  }
}
```
