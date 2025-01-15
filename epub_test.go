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

	chapters := bk.Chapters()
	for _, chapter := range chapters {
		fmt.Printf("title: %+v\n", chapter.Text)

		content, err := bk.ChapterContent(chapter)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("content: %+v\n", string(content))
	}
}
