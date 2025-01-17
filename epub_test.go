package epub

import (
	"fmt"
	"testing"
)

func TestReader(t *testing.T) {
	bk, err := Open("./data/test.epub")
	if err != nil {
		t.Fatal(err)
	}
	defer bk.Close()

	chapters := bk.NavPoints()
	for _, chapter := range chapters {
		fmt.Printf("title: %s\n", chapter.Text)

		content := bk.NavPointContent(chapter)
		fmt.Printf("content: %s\n", content)

		txt := XmlToTxt(content)
		fmt.Printf("txt: %s\n", txt)
	}
}
