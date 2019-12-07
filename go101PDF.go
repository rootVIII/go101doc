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
	"sync"
)

// GoDOC -Build a Go 101 ebook in the current directory.
type GoDOC interface {
	pageRequest(url string) []byte
	setLinks(bookLinks [][]byte)
	getBookData()
	getDecompBuffer() []byte
	gzipWrite(path []byte, out chan<- struct{}, lock *sync.Mutex)
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

func (doc *docMaker) gzipWrite(path []byte, out chan<- struct{}, lock *sync.Mutex) {
	resp := doc.pageRequest(string(path))
	lock.Lock()
	comp := gzip.NewWriter(&doc.buf)
	comp.Write(path)
	comp.Write([]byte("|"))
	comp.Write(resp)
	comp.Write([]byte("|"))
	comp.Close()
	lock.Unlock()
	out <- struct{}{}
}

func (doc *docMaker) getBookData() {
	ch := make(chan struct{})
	var mutex = &sync.Mutex{}
	for _, urlPath := range doc.links {
		go doc.gzipWrite(urlPath, ch, mutex)
	}
	for i := 0; i < len(doc.links); i++ {
		<-ch
	}
}

func (doc *docMaker) getDecompBuffer() []byte {
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
	fmt.Printf("%q\n", goDOC.getDecompBuffer())
}
