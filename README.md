# Sunnified Web Framework

## Using the framework

~~~ go
import (
	"github.com/zaolab/sunnified"
	"net/http"
)

func main() {
	app := sunnified.NewSunnyApp()
	app.Handle("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})
	
	ss.Run(nil)
}
~~~
The Run function accepts certain basic params to configure how the app is being ran.
~~~ go
// all parameters are optional
map[string]interface{}{
	"ip": "string",	// The ip address to bind to.
	"port": "string", // The port to bind the server to. e.g. "8080". Do not include a colon in front.
	"dev": false, // When true, it is equivalent to ip: "127.0.0.1", port: "8080".
	"timeout": time.Duration, // The amount of time till the server times out a request.
	"fcgi": false, // Whether to run it as a fastcgi server instead of a http server
	"sock": false, // Whether to use a unix sock for the fastcgi. Default file created is at /tmp/sunnyapp.sock
	"sockfile": "string", // The custom filename for the unix sock.
}
~~~

---

## Routing/Handlers

Routing offers the same functionality as in net/http, but we included several other functionalities.

Variables are allowed in the path; just wrap them around curly braces.
To access the variable value, you will need to pass in a handler function with *web.Context as its argument.
The value can then be access through the *web.Context object.
The Context type is in the github.com/zaolab/sunnified/web package

~~~ go
import (
	"github.com/zaolab/sunnified"
	"github.com/zaolab/sunnified/web"
	"net/http"
)

func main() {
	app := sunnified.NewSunnyApp()
	app.Handle("/users/{id}", func(context *web.Context) {
		// path variable values are stored in context.PData
		// the PData is a map[string]string with additional functions to get casted values.
		id := context.PData.String("id)
		context.Response.Write([]byte("My ID is" + id))
	})
	
	ss.Run(nil)
}
~~~

Variables can be optional. Put an asterisk symbol after the variable name to mark it as optional.

~~~ go
// this will match /users and also /users/1
app.Handle("/users/{id*}", func(context *web.Context) {
	id := context.PData.String("id")
	if id == "" {
		context.Response.Write([]byte("No ID given"))
	} else {
		context.Response.Write([]byte("My ID is" + id))
	}
})
~~~

Variables can have type. For now, there's only 4 types for variables (int, int64, float, float64).
Use a colon after the variable name and after the asterisk if any, and insert the type.

~~~ go
// this will match /users and also /users/1 but not /users/me
app.Handle("/users/{id*:int}", func(context *web.Context) {
	id := context.PData.Int("id")
	if id == "" {
		context.Response.Write([]byte("No ID given"))
	} else {
		context.Response.Write([]byte("My ID is" + strconv.Itoa(id)))
	}
})
~~~

Variables can be be matched with regular expression. Like typed variables, add a colon and insert the regular expression after the colon.

~~~ go
app.Handle(`/users/name:[a-zA-Z\s]+`, func(context *web.Context) {
	name := context.PData.String("name")
	context.Response.Write([]byte("My Name is" + name))
})
~~~

Path are matched as a whole or as part.
A "/users" path will match "/users" and "/users/" but will not match "/users/something"
But add a trailing slash to the path and it can match path partially.
Unmatched path data can also be accessed through the Context.

~~~ go
// this will math /users, /users/profile, /users/profile/1, etc.
app.Handle("/users/", func(context *web.Context) {
	// path values are stored in context.PData
	// the UPath variable is a []string array with additional functions to get casted values.
	action := context.UPath.String(0)
	id := context.UPath.String(1)
	context.Response.Write([]byte(action))
	context.Response.Write([]byte(id))
})
~~~

Extensions are ignored when matching path.
"/users/profile" will match /users/profile, /users/profile.json, /users/profile.html, etc.
Use the Context to access the extension.

~~~ go
app.Handle("/users/profile", func(context *web.Context) {
	extension := context.Ext
	// codes to send response data base on extension information
})
~~~

You can specifiy which HTTP method that the handler is made to listen to
~~~ go
app.Handle("/users/{id}", func(context *web.Context) {
	// do something here...
}, "GET", "PUT")
~~~

---

## Controllers

The above examples mainly uses the standard net/http compatible function and/or our own context function.
It is also possible to use a struct as a controller which comes with much more functionalities.
The path will be default be /{module*}/{controller*}/{action*}/
The module is the package's name, controller is the controller's name, and action is the function's name

~~~ go
package users

import (
	"github.com/zaolab/sunnified"
	"github.com/zaolab/sunnified/mvc"
	"github.com/zaolab/sunnified/mvc/view"
)

type MyController struct {
	// mvc.BaseController has an embedded Context,
	// which in turn makes your controller inherits all of Context
	// so you can directly access Context data through the controller
	mvc.BaseController
}

// The Construct_ function is called no matter which action is performed.
// It is useful to initial some resources that the controller needs.
func (c *MyController) Construct_(context *web.Context) {
}

// this will match path /users/mycontroller/list
// you can specify HTTP method as a prefix of the function name
// if there is no HTTP method specified, all request will go to it
func (c *MyController) GETList() *view.JsonView {
	result := mvc.VM{
		"users": []string{
			"a",
			"b",
			"c",
		},
		"count": 3,
	}
	return view.NewJsonView(result)
}

// if function name only consists of HTTP method, the action in the path will be ignored...
// e.g. POST /users/mycontroller
// in the path, you can use _ character to denote empty action, and anything after the _ will be in the Context.UPath
// e.g. POST /users/mycontroller/_/path/continues/here
func (c *MyController) POST() *view.JsonView {
	// insert user data into database...
	
	return view.NewJsonView(mvc.VM{"success": true})
}

// Destruct_, like Construct_, is called for every action, but at the end of the request,
// so you can use it to properly close resources initialised in Construct_. 
// *Note* Destruct_() does not have the Context argument
func (c *MyController) Destruct_() {
}

~~~
In the main file
~~~ go
import "users"

// in the main function
// this will creates a default route at /{module*}/{controller*}/{action*}/
// you can keep adding in more controllers, 
// and the app will route based on their package, controller, and function name
app.AddController((*users.MyController)(nil))
~~~

---

## Views
The app allows different form of views to be used in the controller.
In the examples previously, we used the JsonView which outputs response as json data.

If you are not outputting purely json data, you can return map[string]interface{} datatype, 
and the app will use a golang template to render the view.

Unfortunately, the path is hardcoded for the time being.
The templates should be placed in "themes/default/tmpl/{module}/{controller}/" directory.
The file names for each action should be "{action}.{extension}"
Any shared templates to be rendered together with each page template should be placed in "themes/default/tmpl/_share_/".

The default view will render the page based on the extension specified by the end user and find the appropriate template.
e.g. /users/mycontroller/list.html vs /users/mycontroller/list.json
In the first path, the view will look for the list.html template,
and in the second path, the view will look for the list.json template.
If no extension is specified in the path, then .html is used by default.

The view will also perform a gzip compression to the output if the end client supports it.

---

## MiddleWare
The app supports a middleware that are called at 6 different stages (Request, Body, Controller, View, Response, Cleanup)
The Request is called at the start of the request after the route is mapped. This will be useful to initialise database connections, etc.
The Body is called if the request is not HTTP OPTIONS. For OPTIONS request, the chain will end before Body.
The Controller is called if the handler is a app controller instead of just a normal http.Handler.
The View is called before controller's view is rendered. So middleware can inject template data etc. into the view. This will not happen if the handler is not an app controller
The Response is called after the view is rendered/response is written.
The Cleanup is called at the end so that the middleware can perform correct closure to certain resources used.

A custom middleware does not need to provide all 6 functions, but only those that are needed by it.

~~~go
import "github.com/zaolab/sunnified/mware"

type MyMiddleWare struct {
	// embedding the BaseMiddleWare will allow you to only declare the functions that you need
	mware.BaseMiddleWare
}

func (mw MyMiddleWare) Request(context *web.Context) {
}

func (mw MyMiddleWare) Body(context *web.Context) {
}

func (mw MyMiddleWare) Controller(context *web.Context, cm *controller.ControlManager) {
}

func (mw MyMiddleWare) View(context *web.Context, vw mvc.View) {
}

func (mw MyMiddleWare) Response(context *web.Context) {
}

func (mw MyMiddleWare) Cleanup(context *web.Context) {
}
~~~
In the main file
~~~ go
app.AddMiddleWare(MyMiddleWare{})
~~~

Middlewares are called for all the routes in the app router.
To separate middlewares for different routes, you will have to make sub routers.
~~~ go
app.AddMiddleWare(MyMiddleWare{})

subrouter := app.SubRouter("name-of-sub-router")
subrouter.Handle("/i/do/not/want/middleware/for/this/path", func(context *web.Context){})

subrouter2 := app.SubRouter("another-name")
subrouter2.Handle("/i/want/another/middleware/for/this/path", func(context *web.Context){})
subrouter2.AddMiddleWare(MyAnotherMiddleWare{})
~~~

Because each function in the middleware has a context argument, you can easily pass in resources that are needed into the context for controllers to access.
In the middleware...
~~~go
context.SetResource("database", db)
~~~

In the controller...
~~~go
var db *DatabaseType
context.MapResourceValue("database", &db)
// now you can use the db variable to perform whatever task you need it to perform
~~~

Alternatively in the struct declaration of the controller...
~~~go
type MyController struct {
	mvc.BaseController
	// here the app will feed the resource directly into the variable
	// it can be any name but must be an exported field for the app to access
	// the resource must be in the Context before controller initialisation, so preferably before or during Controller middleware function
	DB *DatabaseType `sunnified.res:"database"`
}
~~~