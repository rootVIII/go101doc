package main

// go101PDF.go
// rootVIII

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

// GoPDF -Build a go101 PDF in the current directory.
type GoPDF interface {
	pageRequest(url string) []byte
	setLinks(bookLinks [][]byte)
	getBookData()
	getBufferStr() string
}

type pdfMaker struct {
	GoPDF
	baseURL string
	links   [][]byte
	buf     bytes.Buffer
}

func (pdf pdfMaker) pageRequest(endpoint string) []byte {
	client := &http.Client{}
	req, err := http.NewRequest("GET", pdf.baseURL+endpoint, nil)
	response, err := client.Do(req)
	if err != nil {
		fmt.Printf("request failed: %s\n", pdf.baseURL)
		os.Exit(1)
	}
	//fmt.Printf("%v\n", req)
	rText, _ := ioutil.ReadAll(response.Body)
	return rText
}

func (pdf *pdfMaker) setLinks(bookLinks [][]byte) {
	pdf.links = bookLinks
}

func (pdf *pdfMaker) getBookData() {
	for _, urlPath := range pdf.links {
		resp := pdf.pageRequest(string(urlPath))
		comp := gzip.NewWriter(&pdf.buf)
		comp.Write(urlPath)
		comp.Write([]byte("|"))
		comp.Write(resp)
		comp.Write([]byte("|"))
		comp.Close()
	}
}

func (pdf *pdfMaker) getBufferStr() string {
	return pdf.buf.String()
}

func main() {
	var goPDF GoPDF
	goPDF = &pdfMaker{baseURL: "https://go101.org/article/"}
	mainPage := goPDF.pageRequest("101.html")
	foundLinks := make([][]byte, 0)
	for _, line := range bytes.Split(mainPage, []byte("\n")) {
		if bytes.Contains(line, []byte("<li")) && bytes.Contains(line, []byte("index")) {
			link := bytes.Split(line, []byte("\""))[3]
			foundLinks = append(foundLinks, link)
		}
	}
	goPDF.setLinks(foundLinks)
	goPDF.getBookData()
	fmt.Println(goPDF.getBufferStr())
}
