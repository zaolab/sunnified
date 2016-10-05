package handler

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"text/template"
	"time"
)

const templateFolder = "./errors/"
const templateString = `<!DOCTYPE html>
<html>
<head>
	<title>{{.statuscode}} {{.statustext}}</title>
	<style>
	body, h1, p {
		font-family: Helvetica, Verdana, Arial;
	}
	</style>
</head>
<body>
	<h1>{{.statuscode}} {{.statustext}}</h1>
	<p>An error occured while processing your request. Please try again later.</p>
</body>
</html>`

var (
	NotFoundHandler            = http.HandlerFunc(NotFound)
	InternalServerErrorHandler = http.HandlerFunc(InternalServerError)
	ForbiddenHandler           = http.HandlerFunc(Forbidden)
	templateCache              *template.Template
)

func init() {
	if st, err := os.Stat(templateFolder + "0.html"); err == nil && !st.IsDir() {
		templateCache, _ = template.ParseFiles(templateFolder + "0.html")
	}
	if templateCache == nil {
		templateCache = template.New("0.html")
		templateCache.Parse(templateString)
	}
}

func NewNotFoundHandler() http.Handler {
	return NotFoundHandler
}

func NewInternalServerErrorHandler() http.Handler {
	return InternalServerErrorHandler
}

func NewForbiddenHandler() http.Handler {
	return ForbiddenHandler
}

func NotFound(w http.ResponseWriter, r *http.Request) {
	ErrorHTML(w, r, http.StatusNotFound)
}

func InternalServerError(w http.ResponseWriter, r *http.Request) {
	ErrorHTML(w, r, http.StatusInternalServerError)
}

func Forbidden(w http.ResponseWriter, r *http.Request) {
	ErrorHTML(w, r, http.StatusForbidden)
}

func ErrorHTML(w http.ResponseWriter, r *http.Request, status int) {
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
		}
	}()

	if f, err := os.Open(templateFolder + strconv.Itoa(status) + ".html"); err == nil {
		defer f.Close()

		// we must not send the last-modified header of the file
		// since we do not know the real last-modified of the URI requested (not the error file)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(status)
		// the name "error.html" is not really needed
		// since it is only used to sniff content-type which we already provide
		http.ServeContent(w, r, "error.html", time.Time{}, f)
	} else {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.WriteHeader(status)
		templateCache.Execute(w, map[string]interface{}{"statuscode": status, "statustext": http.StatusText(status)})
	}
}
