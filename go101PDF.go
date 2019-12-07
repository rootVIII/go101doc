package main

// go101doc.go
// rootVIII

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

// GoDOC -Build a Go 101 ebook in the current directory.
type GoDOC interface {
	pageRequest(url string) []byte
	setLinks(bookLinks [][]byte)
	getBookData()
	getBufferStr() []byte
}

type docMaker struct {
	GoDOC
	baseURL string
	links   [][]byte
	buf     bytes.Buffer
}

func (doc docMaker) pageRequest(endpoint string) []byte {
	client := &http.Client{}
	req, err := http.NewRequest("GET", doc.baseURL+endpoint, nil)
	response, err := client.Do(req)
	if err != nil {
		fmt.Printf("request failed: %s\n", doc.baseURL)
		os.Exit(1)
	}
	rText, _ := ioutil.ReadAll(response.Body)
	return rText
}

func (doc *docMaker) setLinks(bookLinks [][]byte) {
	doc.links = bookLinks
}

func (doc *docMaker) getBookData() {
	for _, urlPath := range doc.links {
		resp := doc.pageRequest(string(urlPath))
		comp := gzip.NewWriter(&doc.buf)
		comp.Write(urlPath)
		comp.Write([]byte("|"))
		comp.Write(resp)
		comp.Write([]byte("|"))
		comp.Close()
	}
}

func (doc *docMaker) getBufferStr() []byte {
	readComp, _ := gzip.NewReader(&doc.buf)
	io.Copy(os.Stdout, readComp)
	return doc.buf.Bytes()
}

func main() {
	var goDOC GoDOC
	goDOC = &docMaker{baseURL: "https://go101.org/article/"}
	mainPage := goDOC.pageRequest("101.html")
	foundLinks := make([][]byte, 0)
	for _, line := range bytes.Split(mainPage, []byte("\n")) {
		if bytes.Contains(line, []byte("<li")) && bytes.Contains(line, []byte("index")) {
			link := bytes.Split(line, []byte("\""))[3]
			foundLinks = append(foundLinks, link)
		}
	}
	goDOC.setLinks(foundLinks)
	goDOC.getBookData()
	fmt.Printf("%q\n", goDOC.getBufferStr())
	//output := goDOC.getBufferStr()
}
