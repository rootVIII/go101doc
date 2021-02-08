package main

/*
// go101doc.go
// rootVIII
*/

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

// go101DocMaker builds a Go 101 ebook in the current directory.
type go101DocMaker interface {
	pageRequest(url string) ([]byte, error)
	setLinks(bookLinks [][]byte)
	getLinks() [][]byte
	getBookData()
	getDecompBuffer() ([]byte, error)
	gzipWrite(path []byte, out chan<- struct{}, lock *sync.Mutex)
}

type go101Doc struct {
	baseURL string
	links   [][]byte
	buf     bytes.Buffer
}

func (doc go101Doc) pageRequest(endpoint string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", doc.baseURL+endpoint, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	rBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return rBytes, nil
}

func (doc *go101Doc) setLinks(bookLinks [][]byte) {
	doc.links = bookLinks
}

func (doc *go101Doc) getLinks() [][]byte {
	return doc.links
}

func (doc *go101Doc) gzipWrite(path []byte, out chan<- struct{}, lock *sync.Mutex) {
	resp, err := doc.pageRequest(string(path))
	if err != nil {
		fmt.Printf("Encountered error on %q\n%v\n", path, err)
		out <- struct{}{}
	}

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

func (doc *go101Doc) getBookData() {
	ch := make(chan struct{})
	var mutex = &sync.Mutex{}
	for _, urlPath := range doc.links {
		go doc.gzipWrite(urlPath, ch, mutex)
	}
	for i := 0; i < len(doc.links); i++ {
		<-ch
	}
}

func (doc *go101Doc) getDecompBuffer() ([]byte, error) {
	readComp, err := gzip.NewReader(&doc.buf)
	if err != nil {
		return nil, err
	}
	out, err := ioutil.ReadAll(readComp)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func exitOnError(err error) {

}

func main() {
	var goDoc go101DocMaker
	goDoc = &go101Doc{baseURL: "https://go101.org/article/"}
	mainPage, err := goDoc.pageRequest("101.html")
	if err != nil {
		exitOnError(fmt.Errorf("Unable to reach go101.org"))
	}
	foundLinks := make([][]byte, 0)

	for _, line := range bytes.Split(mainPage, []byte("\n")) {
		if bytes.Contains(line, []byte("<li")) && bytes.Contains(line, []byte("index")) {
			link := bytes.Split(line, []byte("\""))[3]
			foundLinks = append(foundLinks, link)
		}
	}

	goDoc.setLinks(foundLinks)
	goDoc.getBookData()

	decompBuffer, err := goDoc.getDecompBuffer()
	if err != nil {
		exitOnError(err)
	}

	bookData := bytes.Split(decompBuffer, []byte("|+|"))
	cwd, err := os.Getwd()
	if err != nil {
		exitOnError(err)
	}

	path := cwd + string(os.PathSeparator) + "go101.html"
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		exitOnError(err)
	}

	defer f.Close()
	for _, pageName := range goDoc.getLinks() {
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
