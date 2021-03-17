package epub

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/beevik/etree"
	"github.com/moraes/isbn"
	"golang.org/x/tools/godoc/vfs/zipfs"
)

// Epub represents a epub type book
type Epub struct {
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	Publisher   string    `json:"publisher"`
	Language    string    `json:"language"`
	HasCover    bool      `json:"has_cover"`
	ISBN        string    `json:"isbn"`
	Series      string    `json:"series"`
	SeriesIndex float64   `json:"series_index"`
	Description string    `json:"description"`
	PublishDate time.Time `json:"publish_date"`
}

// ParseFile takes a filepath and returns an Epub if possible
func ParseFile(bookpath string) (bk *Epub, cover []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			bk = nil
			err = fmt.Errorf("Unknown error parsing book. Skipping. Error: %s", r)
		}
	}()

	book := new(Epub)
	book.Language = ""
	book.Title = filepath.Base(bookpath)

	zr, err := zip.OpenReader(bookpath)
	if err != nil {
		return
	}

	zfs := zipfs.New(zr, "epub")

	rsk, err := zfs.Open("/META-INF/container.xml")
	if err != nil {
		return
	}
	defer rsk.Close()
	container := etree.NewDocument()
	_, err = container.ReadFrom(rsk)
	if err != nil {
		return
	}
	rootfile := ""
	for _, e := range container.FindElements("//rootfiles/rootfile[@full-path]") {
		rootfile = e.SelectAttrValue("full-path", "")
	}
	if rootfile == "" {
		err = errors.New("Cannot parse container")
		return
	}

	rootReadSeeker, err := zfs.Open("/" + rootfile)
	if err != nil {
		return
	}
	defer rootReadSeeker.Close()
	opf := etree.NewDocument()
	_, err = opf.ReadFrom(rootReadSeeker)
	if err != nil {
		return
	}

	opfDir := filepath.Dir(rootfile)

	book.Title = filepath.Base(bookpath)
	for _, e := range opf.FindElements("//title") {
		book.Title = e.Text()
		break
	}
	for _, e := range opf.FindElements("//creator") {
		book.Author = e.Text()
		break
	}
	for _, el := range opf.FindElements("//publisher") {
		book.Publisher = el.Text()
		break
	}
	for _, e := range opf.FindElements("//description") {
		book.Description = e.Text()
		break
	}
	for _, e := range opf.FindElements("//language") {
		book.Language = e.Text()
		break
	}

	isbnTags := []string{
		"//source",
		"//identifier",
	}

findISBN:
	for _, tag := range isbnTags {
		for _, el := range opf.FindElements(tag) {
			val := el.Text()
			if len(val) < 10 {
				continue
			}
			if val[0:9] == "urn:isbn:" {
				val = val[9:]
			}

			if isbn.Validate(val) {
				book.ISBN = val
				break findISBN
			}
		}
	}

	pubDate := ""
	for _, el := range opf.FindElements("//date") {
		event := el.SelectAttrValue("opf:event", "")
		if event == "original-publication" || event == "published" || event == "publication" {
			pubDate = el.Text()
			// found a concrete publication date; we're done
			break
		} else if event == "" {
			pubDate = el.Text()
			// keep searching in case we can find a date specifically tagged as a publication date
		}
	}

	coverpath := ""
	for _, el := range opf.FindElements("//meta[@name='cover']") {
		coverid := el.SelectAttrValue("content", "")
		if coverid != "" {
			for _, f := range opf.FindElements("//[@id='" + coverid + "']") {
				coverpath = f.SelectAttrValue("href", "")
				if coverpath != "" {
					coverpath = "/" + opfDir + "/" + coverpath
					break
				}
			}
			break
		}
	}

	if coverpath != "" {
		cover = func() []byte {

			cr, err := zfs.Open(coverpath)

			if err != nil {
				return nil
			}
			var b bytes.Buffer
			defer cr.Close()
			i, _, err := image.Decode(cr)
			if err != nil {
				return nil
			}
			err = jpeg.Encode(&b, i, nil)
			if err != nil {
				return nil
			}
			return b.Bytes()
		}()

		if len(cover) > 0 {
			book.HasCover = true
		}
	}

	book.PublishDate = parsePublishDate(pubDate)

	// Calibre series metadata
	if el := opf.FindElement("//meta[@name='calibre:series']"); el != nil {
		book.Series = el.SelectAttrValue("content", "")

		if el := opf.FindElement("//meta[@name='calibre:series_index']"); el != nil {
			book.SeriesIndex, _ = strconv.ParseFloat(el.SelectAttrValue("content", "0"), 64)
		}

	}

	// EPUB3 series metadata
	if book.Series == "" {
		if el := opf.FindElement("//meta[@property='belongs-to-collection']"); el != nil {
			book.Series = strings.TrimSpace(el.Text())

			var ctype string
			if id := el.SelectAttrValue("id", ""); id != "" {
				for _, el := range opf.FindElements("//meta[@refines='#" + id + "']") {
					val := strings.TrimSpace(el.Text())
					switch el.SelectAttrValue("property", "") {
					case "collection-type":
						ctype = val
					case "group-position":
						book.SeriesIndex, _ = strconv.ParseFloat(val, 64)
					}
				}
			}

			if ctype != "" && ctype != "series" {
				book.Series, book.SeriesIndex = "", 0
			}
		}
	}

	parts := strings.Split(book.Series, "#")
	if len(parts) == 2 {
		book.SeriesIndex, _ = strconv.ParseFloat(parts[1], 64)
	}

	book.Series = strings.TrimSuffix(book.Series, fmt.Sprintf("#%.0f", book.SeriesIndex))
	book.Series = strings.TrimSuffix(book.Series, fmt.Sprintf("#%.1f", book.SeriesIndex))
	book.Series = strings.TrimSpace(book.Series)

	return book, cover, nil

}

func parsePublishDate(s string) time.Time {
	// handle the various dumb decisions people make when encoding dates
	format := ""
	switch len(s) {
	case 32:
		//2012-02-13T20:20:58.175203+00:00
		format = "2006-01-02T15:04:05.000000-07:00"
	case 25:
		//2000-10-31 00:00:00-06:00
		//2009-04-19T22:00:00+00:00
		format = "2006-01-02" + string(s[10]) + "15:04:05-07:00"
	case 20:
		//2016-08-11T14:09:25Z
		format = "2006-01-02T15:04:05Z"
	case 19:
		//2008-01-28T07:00:00
		//2000-10-31 00:00:00
		format = "2006-01-02" + string(s[10]) + "15:04:05"
	case 10:
		//1998-07-01
		format = "2006-01-02"
	default:
		return time.Time{}
	}

	t, err := time.Parse(format, s)
	if err != nil {
		t = time.Time{}
	}
	return t
}
