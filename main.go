package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func FindAudioLink(link string) ([]string, error) {
	log.Printf("Looking for audio link: %s", link)
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "curl")
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("Failed to GET %s - %d", link, res.StatusCode))
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	results := make([]string, 0)
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		download, found := s.Attr("href")
		if !found {
			return
		}
		if strings.Contains(download, ".gme") {
			fmt.Printf("Downloadlink %d: '%s'\n", i, download)
			results = append(results, download)
		}
	})
	if len(results) == 0 {
		return nil, errors.New("Couldn't find download link")
	}
	return results, nil
}

func FindProductsFromService() []string {
	link := "https://service.ravensburger.de/tiptoi%C2%AE/tiptoi%C2%AE_Audiodateien"
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", "curl")
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	results := make([]string, 0)
	doc.Find(".mt-show-more-listing").Each(func(i int, s *goquery.Selection) {
		rlink, _ := s.Find("a").Attr("href")
		rname, _ := s.Find("a").Attr("title")
		fmt.Printf("Result %d: %s - %s\n", i, rname, rlink)
		results = append(results, rlink)
	})
	return results
}

func DownloadFile(link string, destinationDir string) error {
	linkurl, err := url.Parse(link)
	if err != nil {
		return err
	}
	filename := path.Base(linkurl.Path)
	file, err := os.Create(path.Join(destinationDir, filename))
	if err != nil {
		return err
	}
	defer file.Close()

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	res, err := http.DefaultClient.Get(link)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	_, err = io.Copy(file, res.Body)
	log.Printf("written file to %s", file.Name())

	return err
}

func main() {
	links := FindProductsFromService()
	downloads, err := FindAudioLink(links[0])
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("found files: %s", downloads)
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	err = DownloadFile(downloads[0], home+"/Downloads")
	if err != nil {
		log.Fatal(err)
	}
}
