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

type Item struct {
	name       string
	detailLink string
}

// for list.Item
func (i Item) FilterValue() string {
	return i.name
}

// for list.DefaultItem
func (i Item) Title() string {
	return i.name
}

// for list.DefaultItem
func (i Item) Description() string {
	return ""
}

func load(link string) (io.ReadCloser, error) {
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
	if res.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("Failed to GET %s - %d", link, res.StatusCode))
	}
	return res.Body, nil
}

func FindAudioLink(link string) ([]Item, error) {
	//log.Printf("Looking for audio link: %s", link)
	body, err := load(link)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, err
	}

	results := make([]Item, 0)
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		download, found := s.Attr("href")
		if !found {
			return
		}
		if strings.Contains(download, ".gme") {
			//fmt.Printf("Downloadlink %d: '%s'\n", i, download)
			results = append(results, Item{name: download, detailLink: download})
		}
	})
	if len(results) == 0 {
		return nil, errors.New("Couldn't find download link")
	}
	return results, nil
}

func FindProductsFromService() []Item {
	link := "https://service.ravensburger.de/tiptoi%C2%AE/tiptoi%C2%AE_Audiodateien"
	body, err := load(link)
	if err != nil {
		log.Fatal(err)
	}
	defer body.Close()

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		log.Fatal(err)
	}

	results := make([]Item, 0)
	doc.Find(".mt-show-more-listing").Each(func(i int, s *goquery.Selection) {
		rlink, _ := s.Find("a").Attr("href")
		rname, _ := s.Find("a").Attr("title")
		//fmt.Printf("Result %d: %s - %s\n", i, rname, rlink)

		results = append(results, Item{rname, rlink})
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

	body, err := load(link)
	if err != nil {
		log.Fatal(err)
	}
	defer body.Close()

	_, err = io.Copy(file, body)
	//log.Printf("written file to %s", file.Name())

	return err
}
