package main

// go101doc.go
// rootVIII

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
)

// GoDOC -Build a Go 101 ebook in the current directory.
type GoDOC interface {
	pageRequest(url string) []byte
	setLinks(bookLinks [][]byte)
	getLinks() [][]byte
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

func (doc *docMaker) getLinks() [][]byte {
	return doc.links
}

func (doc *docMaker) gzipWrite(path []byte, out chan<- struct{}, lock *sync.Mutex) {
	resp := doc.pageRequest(string(path))
	var respMinusFooter []byte
	for i := range resp {
		if resp[i+1] == byte('h') && resp[i+2] == byte('r') && resp[i+3] == byte('>') {
			break
		} else {
			respMinusFooter = append(respMinusFooter, resp[i])
		}
	}
	lock.Lock()
	comp := gzip.NewWriter(&doc.buf)
	comp.Write(path)
	comp.Write(respMinusFooter)
	comp.Write([]byte("</div></body></html>"))
	comp.Write([]byte("|+|"))
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
	out, _ := ioutil.ReadAll(readComp)
	return out
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
	bookData := bytes.Split(goDOC.getDecompBuffer(), []byte("|+|"))
	cwd, _ := os.Getwd()
	path := cwd + string(os.PathSeparator) + "go101.html"
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()
	for _, pageName := range goDOC.getLinks() {
		for _, pageData := range bookData {
			current := pageData[:len(pageName)]
			if bytes.Compare(pageName, current) != 0 {
				continue
			}
			page := html.UnescapeString(string(pageData[len(pageName):len(pageData)]))
			f.WriteString(page)
		}
	}
	fmt.Printf("File created:  %s\n", path)
}
