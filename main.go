package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/koron/go-dproxy"
)

var re = regexp.MustCompile(`(.*)\((.*)\) - FANZA同人`)

// FanzaAPIInfo includes api information for FANZA
type FanzaAPIInfo struct {
	APIID       string `json:"api_id"`
	AffiliateID string `json:"affiliate_id"`
}

// Work defines ...
type Work struct {
	Title string
	Maker string
}

func (w *Work) filename() string {
	return "[" + string(w.Maker) + "] " + string(w.Title) + ".zip"
}

func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func walkNonRecursive(root string, walkFn filepath.WalkFunc) error {
	fInfo, _ := os.Stat(root)
	if fInfo.IsDir() {
		root = filepath.Join(root, "*")
	}

	files, err := filepath.Glob(root)
	if err != nil {
		return err
	}
	for _, f := range files {
		info, err := os.Lstat(f)
		if err := walkFn(f, info, err); err != nil {
			return err
		}
	}
	return nil
}

func getWorkInfo(path string, info os.FileInfo, err error, fanzaAPIInfo *FanzaAPIInfo, enableRename bool) error {
	if err != nil {
		return err
	} else if info.IsDir() {
		return nil
	}
	if strings.HasPrefix(info.Name(), "d_") && strings.HasSuffix(info.Name(), ".zip") {
		return getWorkInfoFromFanza(path, info.Name()[:8], fanzaAPIInfo, enableRename)
	} else if strings.HasPrefix(info.Name(), "RJ") && strings.HasSuffix(info.Name(), ".zip") {
		return getWorkInfoFromDLsite(path, info.Name()[:8], enableRename)
	}
	return nil
}

func getWorkInfoFromDLsite(path, contentID string, enableRename bool) error {
	html, err := loadHTMLFromWeb("https://www.dlsite.com/maniax/work/=/product_id/" + string(contentID) + ".html")
	if err != nil {
		return err
	}
	doc, err := goquery.NewDocumentFromReader(html)
	if err != nil {
		return err
	}
	work := &Work{}
	work.Title = doc.Find("h1#work_name>a").Text()
	work.Maker = doc.Find("span.maker_name>a").Text()
	fmt.Printf("%v -> %v\n", path, work.filename())
	if enableRename {
		return os.Rename(path, filepath.Join(filepath.Dir(path), work.filename()))
	}
	return nil
}

func getWorkInfoFromFanza(path, contentID string, fanzaAPIInfo *FanzaAPIInfo, enableRename bool) error {
	reqURL := generateRequestURL(contentID, fanzaAPIInfo)
	resp, err := http.Get(reqURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Printf("%v -> ", path)

	var v interface{}
	var err1 error
	json.Unmarshal(respBytes, &v)
	work := &Work{}
	work.Title, err1 = dproxy.New(v).M("result").M("items").A(0).M("title").String()
	work.Maker, err = dproxy.New(v).M("result").M("items").A(0).M("iteminfo").M("maker").A(0).M("name").String()
	if err1 != nil || err != nil {
		work, err = scrapeFanza(path, contentID, enableRename)
		if err != nil {
			return err
		}
	}
	fmt.Println(work.filename())
	if enableRename {
		return os.Rename(path, filepath.Join(filepath.Dir(path), work.filename()))
	}
	return nil
}

func loadHTMLFromWeb(url string) (io.Reader, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(body), nil
}

func scrapeFanza(path, contentID string, enableRename bool) (*Work, error) {
	html, err := loadHTMLFromWeb("https://www.dmm.co.jp/dc/doujin/-/detail/=/cid=" + string(contentID))
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(html)
	if err != nil {
		return nil, err
	}
	titleRaw := doc.Find("title").Text()
	result := re.FindAllStringSubmatch(titleRaw, -1)
	if len(result[0]) != 3 {
		return nil, fmt.Errorf("Error during pattern match")
	}
	return &Work{result[0][1], result[0][2]}, nil
}
func generateRequestURL(contentID string, fanzaAPIInfo *FanzaAPIInfo) string {
	b := &strings.Builder{}
	b.WriteString("https://api.dmm.com/affiliate/v3/ItemList?api_id=")
	b.WriteString(fanzaAPIInfo.APIID)
	b.WriteString("&affiliate_id=")
	b.WriteString(fanzaAPIInfo.AffiliateID)
	b.WriteString("&site=FANZA&cid=")
	b.WriteString(contentID)
	return b.String()
}

func readFanzaAPIInfo(path string) (*FanzaAPIInfo, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var info FanzaAPIInfo
	if err := json.Unmarshal(content, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func main() {
	var walk func(string, filepath.WalkFunc) error
	var enableRecursive bool
	var enableRename bool
	var fanzaAPIInfoPath string
	flag.BoolVar(&enableRecursive, "r", false, "Visit files recursively")
	flag.StringVar(&fanzaAPIInfoPath, "s", "settings.json", "Path to settings.json")
	flag.BoolVar(&enableRename, "e", false, "Execute renaming")
	flag.Parse()
	args := flag.Args()

	fanzaAPIInfo, err := readFanzaAPIInfo(fanzaAPIInfoPath)
	if err != nil {
		log.Fatalln(err)
	}

	if !enableRecursive {
		walk = walkNonRecursive
	} else {
		walk = filepath.Walk
	}
	walkFn := func(path string, info os.FileInfo, err error) error {
		return getWorkInfo(path, info, err, fanzaAPIInfo, enableRename)
	}

	for _, fileOrDir := range args {
		fileOrDir := filepath.Clean(fileOrDir)
		if !exists(fileOrDir) {
			fmt.Printf("%v does not exist\n", fileOrDir)
			continue
		}
		if err := walk(fileOrDir, walkFn); err != nil {
			log.Fatalln(err)
		}
	}
}
