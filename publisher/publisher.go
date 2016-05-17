package publisher

import (
	htemplate "html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const DEFAULT_VIEWPATH = "views/"

type CTemplate interface {
	Execute(wr io.Writer, data interface{}) error
	ExecuteTemplate(wr io.Writer, name string, data interface{}) error
	Name() string
}

type Publisher interface {
	Publish(wr io.Writer, name string, data interface{}) error
}

type Renderer func(name string, data interface{}) ([]byte, error)

type SunnyPublisher struct {
	renderer map[string]Renderer
	htmpl    *htemplate.Template
	tmpl     *template.Template
	tchannel chan []CTemplate
	fmap     map[string]interface{}
}

func NewSunnyPublisher(p string) *SunnyPublisher {
	var htmpl *htemplate.Template = htemplate.New("index.html")
	var tmpl *template.Template = template.New("index")
	var plen int = 0

	if strings.Contains(p, `\`) {
		p = strings.Replace(p, `\`, "/", -1)
	} else if p == "" {
		p = DEFAULT_VIEWPATH
	}

	if plen = len(p); p[plen-1] != "/" {
		p = p + "/"
		plen += 1
	}

	filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			b, err := ioutil.ReadFile(path)

			if err == nil {
				if strings.HasSuffix(info.Name(), ".html") {
					htmpl.New(path[plen:]).Parse(string(b))
				} else {
					tmpl.New(path[plen:]).Parse(string(b))
				}
			}
		}

		return nil
	})

	return &SunnyPublisher{
		renderer: make(map[string]Renderer),
		htmpl:    htmpl,
		tmpl:     tmpl,
	}
}

// if template can't be found and there exists a publisher for the specific ext,
// the publisher will be used to render and publish the content
func (p *SunnyPublisher) AddRenderer(ext string, renderer Renderer) {
	// TODO: add mutex
	p.renderer[ext] = renderer
}

func (p *SunnyPublisher) Publish(wr io.Writer, name string, data interface{}) {
	// TODO: use sync.Pool instead once GAE supports 1.3

	// get cloned template from pool
	// add contextual funcmap to the template

	if strings.HasSuffix(name, ".html") {
		ht, err := p.htmpl.Clone()
		ht.Funcs(htemplate.FuncMap(p.fmap))

		if err == nil {
			ht = ht.Lookup(name)
			if ht != nil {
				ht.Execute(wr, data)
			} else {
				goto renderer
			}
		} else {
			goto renderer
		}
	} else {
		ht, err := p.tmpl.Clone()
		ht.Funcs(template.FuncMap(p.fmap))

		if err == nil {
			if ht != nil {
				ht.Execute(wr, data)
			} else {
				goto renderer
			}
		} else {
			goto renderer
		}
	}

	return

renderer:
	ext := ".html"
	b, err := p.renderer[ext](name, data)
	if err == nil {
		wr.Write(b)
	}
}

func FreePublisher(t CTemplate) {

}
