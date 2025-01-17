package epub

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	ht "golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

type HTML struct {
	links    map[string]bool
	iframes  map[string]bool
	images   map[string]bool
	eraseCnt int
	output   *bytes.Buffer
	curUrl   *url.URL
	lastEol  bool
}

type GetOpts struct {
	Timeout  time.Duration     // request timeout
	Agent    string            // user agent
	Headers  map[string]string // request header
	NoDecode bool              // decode to utf-8 if charset is not utf-8
}

var (
	ErrGetFailed error

	tagErase map[string]bool
	skpUrls  map[string]bool
	defOpts  *GetOpts
	bShy     []byte
	hTags    map[string]bool
)

func init() {

	ErrGetFailed = errors.New("get request failed")

	tagErase = map[string]bool{}

	for _, tg := range []string{"audio", "del", "form", "iframe", "link", "meta", "noscript", "option", "s",
		"script", "select", "source", "strike", "style", "svg", "title", "video"} {
		tagErase[tg] = true
	}

	hTags = make(map[string]bool)

	for _, tg := range []string{"h1", "h2", "h3", "h4", "h5", "h6"} {
		hTags[tg] = true
	}

	skpUrls = map[string]bool{}

	for _, href := range []string{"", "#", "javascript:void(0);"} {
		skpUrls[href] = true
	}

	bShy = []byte{194, 173}

	defOpts = &GetOpts{
		Timeout:  time.Second * 5,
		Agent:    "html/bot",
		Headers:  make(map[string]string),
		NoDecode: false,
	}
}

func New() *HTML {
	res := &HTML{}

	res.links = make(map[string]bool)
	res.iframes = make(map[string]bool)
	res.images = make(map[string]bool)

	res.output = bytes.NewBuffer(nil)

	return res
}

func (h *HTML) ParseString(str string) {
	h.Parse(strings.NewReader(str))
}

func (h *HTML) beforeParse() {

	h.links = make(map[string]bool)
	h.iframes = make(map[string]bool)
	h.images = make(map[string]bool)

	h.eraseCnt = 0
	h.lastEol = true

	h.output.Reset()
}

func (h *HTML) Parse(r io.Reader) {

	h.beforeParse()

	parser := ht.NewTokenizer(r)

	for {
		tt := parser.Next()

		switch {

		case tt == ht.ErrorToken:
			return

		case tt == ht.StartTagToken:
			t := parser.Token()
			h.onStartTag(&t)

		case tt == ht.EndTagToken:
			t := parser.Token()
			h.onCloseTag(&t)

		case tt == ht.SelfClosingTagToken:
			t := parser.Token()
			h.onStartTag(&t)
			h.onCloseTag(&t)

		case tt == ht.TextToken:
			h.onText(parser.Text())

		}
	}
}

func (h *HTML) onStartTag(t *ht.Token) {

	if tagErase[t.Data] {
		if t.Data != "meta" && t.Data != "link" {
			h.eraseCnt++
		}
	}

	if hTags[t.Data] {
		h.output.WriteRune('\n')
	}

	switch t.Data {

	case "iframe":

		for _, a := range t.Attr {
			if a.Key == "src" {
				h.iframes[a.Val] = true
			}
		}

	case "img":

		for _, a := range t.Attr {
			if a.Key == "src" {
				h.images[a.Val] = true
			}
		}

	case "a":

		for _, a := range t.Attr {
			if a.Key == "href" {
				h.links[a.Val] = true
			}
		}
	}

	if h.eraseCnt > 0 {
		return
	}

	switch t.Data {

	case "br":

		if !h.lastEol {
			h.output.WriteRune('\n')
			h.lastEol = true
		}

	case "wbr":

		h.lastEol = true

	}
}

func (h *HTML) onCloseTag(t *ht.Token) {

	if tagErase[t.Data] && t.Data != "meta" && t.Data != "link" {
		h.eraseCnt--
		if h.eraseCnt < 0 {
			h.eraseCnt = 0
		}
	}

	if h.eraseCnt > 0 {
		return
	}

	if hTags[t.Data] {
		if !h.lastEol {
			h.output.WriteString("\n\n")
			h.lastEol = true
		}
		return
	}

	if t.Data == "p" || t.Data == "div" || t.Data == "ol" || t.Data == "ul" {
		if !h.lastEol {
			h.output.WriteRune('\n')
			h.lastEol = true
		}
	}
}

func (h *HTML) onText(data []byte) {
	if h.eraseCnt == 0 {

		data = bytes.TrimSpace(data)
		if len(data) == 0 {
			return
		}

		if !h.lastEol {
			h.output.WriteRune(' ')
		}
		h.lastEol = false

		for {
			idx := bytes.Index(data, bShy)

			if idx < 0 {
				h.output.Write(data)
				break
			}

			h.output.Write(data[:idx])
			data = data[idx+2:]
		}

	}
}

func (h *HTML) Text() []byte {
	return h.output.Bytes()
}

func (h *HTML) EachLink(callback func(string)) {
	for ul := range h.links {
		if skpUrls[ul] {
			continue
		}

		if ul[0] == '#' || strings.Index(ul, "mailto:") == 0 || strings.Index(ul, "javascript:") == 0 {
			continue
		}
		callback(h.prepareUrl(ul))
	}
}

func (h *HTML) EachImage(callback func(string)) {
	for src := range h.images {
		callback(h.prepareUrl(src))
	}
}

func (h *HTML) EachIframe(callback func(string)) {
	for src := range h.iframes {
		callback(h.prepareUrl(src))
	}
}

func (h *HTML) prepareUrl(src string) string {

	if h.curUrl == nil {
		return src
	}

	if strings.Index(src, "//") == 0 {
		return h.curUrl.Scheme + ":" + src
	}

	if strings.Index(src, "/") == 0 {
		return h.curUrl.Scheme + "://" + h.curUrl.Hostname() + src
	}

	return src
}

func (h *HTML) SetUrl(pageUrl string) error {
	purl, err := url.Parse(pageUrl)
	if err != nil {
		return err
	}
	h.curUrl = purl
	return nil
}

func (h *HTML) ResetUrl() {
	h.curUrl = nil
}

func (h *HTML) Get(pageUrl string, opts *GetOpts) error {

	h.SetUrl(pageUrl)

	if opts == nil {
		opts = defOpts
	}

	tr := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
	}

	ua := &http.Client{
		Timeout:   opts.Timeout,
		Transport: tr,
	}

	req, err := http.NewRequest("GET", pageUrl, nil)
	if err != nil {
		return err
	}

	for k, v := range opts.Headers {
		req.Header.Set(k, v)
	}

	if opts.Agent != "" {
		req.Header.Set("User-Agent", opts.Agent)
	}

	resp, err := ua.Do(req)
	if err != nil || resp == nil {
		return ErrGetFailed
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ErrGetFailed
	}

	var text []byte

	if opts.NoDecode {
		text, err = io.ReadAll(resp.Body)
		if err != nil {
			return ErrGetFailed
		}
	} else {
		utf8, err1 := charset.NewReader(resp.Body, resp.Header.Get("Content-Type"))
		if err1 != nil {
			return ErrGetFailed
		}

		text, err = io.ReadAll(utf8)
		if err != nil {
			return ErrGetFailed
		}
	}

	h.Parse(bytes.NewReader(text))

	return nil
}

func XmlToTxt(xml string) string {
	h := New()
	h.ParseString(xml)
	return string(h.Text())
}
