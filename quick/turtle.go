package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func rdfToURL(rdf string) (string, string) {
	if strings.HasPrefix(rdf, "ns:") {
		return "https://www.googleapis.com/freebase/v1/rdf/" + strings.Replace(rdf[3:], ".", "/", -1), ""
	} else if strings.HasPrefix(rdf, "<") {
		return rdf[1 : len(rdf)-1], ""
	} else if strings.HasPrefix(rdf, "\"") {
		i := strings.LastIndex(rdf, "@")
		var body, lang string
		if i >= 0 {
			body = rdf[1 : i-1]
			lang = rdf[i+1:]
		} else {
			body = rdf[1 : len(rdf)-1]
		}
		body, _ = url.QueryUnescape(body)
		return body, lang
	}
	return "", ""
}

func main() {
	//https://www.googleapis.com/freebase/v1/rdf/m/02mjmr

	key := os.Args[len(os.Args)-1]
	url := "https://www.googleapis.com/freebase/v1/rdf" + key
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "    ns:") || strings.HasPrefix(line, "    ns:rdf:") {
			continue
		}
		line = strings.Trim(line, " \t\n;")
		bits := strings.Split(line, "    ")
		pred, _ := rdfToURL(bits[0])
		obj, lang := rdfToURL(bits[1])
		log.Printf("Uploading Pred: %s, Obj: %s, Lang: %s", pred, obj, lang)
		uploadTriple(url, pred, obj, lang)
	}
}

func uploadTriple(subj, pred, obj, lang string) {
	resp, err := http.PostForm("http://localhost:8080/api/v1/insert",
		url.Values{"subj": {subj}, "pred": {pred}, "obj": {obj}, "lang": {lang}})

	if nil != err {
		fmt.Println("errorination happened getting the response", err)
		return
	}

	defer resp.Body.Close()
	fmt.Println(ioutil.ReadAll(resp.Body))
}
