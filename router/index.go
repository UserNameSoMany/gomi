package route

import (
	"gomi/iType"
)

var (
	//POST ...
	POST = "POST"

	//PUT ...
	PUT = "PUT"

	//GET ...
	GET = "GET"

	//DELETE ...
	DELETE = "DELETE"
)

//Router ...
type Router struct {
	prefix string
	middle []iType.Middle
	route  *route
}

type route struct {
	label        byte
	children     childrenRoute
	parent       *route
	prefix       string
	regexMatcher string
	handler      *handler
	root         *Router
}

type handler struct {
	get    iType.BindMiddle
	post   iType.BindMiddle
	put    iType.BindMiddle
	delete iType.BindMiddle
}

type childrenRoute []*route

//New ...
func New(prefix string) *Router {
	router := &Router{
		prefix: prefix,
	}
	route := route{
		root: router,
	}
	router.route = &route
	return router
}

//Use ...
func (r *Router) Use(handler iType.Middle) {
	r.middle = append(r.middle, handler)
}

//Get ...
func (r *Router) Get(path string, handler ...iType.Middle) {
	r.route.add(GET, path, handler...)
}

//Post ...
func (r *Router) Post(path string, handler ...iType.Middle) {
	r.route.add(POST, path, handler...)
}

//Put ...
func (r *Router) Put(path string, handler ...iType.Middle) {
	r.route.add(PUT, path, handler...)
}

//Delete ...
func (r *Router) Delete(path string, handler ...iType.Middle) {
	r.route.add(DELETE, path, handler...)
}

//Route ...
func (r *Router) Route() iType.Middle {
	return func(ctx *iType.Ctx, bind iType.BindMiddle) error {
		handler := r.search(ctx)
		if handler == nil {
			return bind(ctx)
		}
		err := handler(ctx)
		if err != nil {
			return err
		}
		return bind(ctx)
	}
}

func (r *Router) search(ctx *iType.Ctx) iType.BindMiddle {
	req := ctx.Req
	path := req.URL.Path
	method := req.Method
	return findHandlerByMethodAndPath(r.route, method, path)
}

func findHandlerByMethodAndPath(r *route, method, path string) iType.BindMiddle {
	for {
		if r == nil {
			return nil
		}
		l := 0
		prefix := r.prefix
		pathLength := len(path)
		preLength := len(prefix)
		max := preLength
		if max > pathLength {
			max = pathLength
		}
		for ; l < max && prefix[l] == path[l]; l++ {
		}
		if path[l:] == "" {
			return r.getHandlerByMethod(method)
		}
		path = path[l:]
		cr := findRouteByLabel(r, path[0])
		if cr != nil {
			r = cr
			continue
		}
		return nil
	}
}

func (r *route) add(method, path string, handler ...iType.Middle) {
	if len(handler) == 0 {
		handler = append(handler, func(ctx *iType.Ctx, b iType.BindMiddle) error { return nil })
	}
	handler = append(r.root.middle, handler...)
	middle := iType.ExtendMiddleSlice(handler)
	for {
		prefix := r.prefix
		prefixLength := len(prefix)
		pathLength := len(path)
		max := prefixLength
		if max > pathLength {
			max = pathLength
		}
		l := 0
		for l = 0; l < max && prefix[l] == path[l]; l++ {
		}
		if l == 0 {
			r.prefix = path
			r.children = nil
			r.label = path[0]
			r.regexMatcher = path
			r.addMethodHandler(method, middle)
		} else if l < prefixLength {
			newPrefix := path[0:l]
			otn := convertToNew(r.prefix[l:], r.children, r.handler)
			r.prefix = newPrefix
			r.children = nil
			r.handler = nil
			r.addChildren(otn)
			path = path[l:]
			if l == pathLength {
				r.addMethodHandler(method, middle)
			} else {
				newRoute := &route{
					label:        path[0],
					prefix:       path,
					regexMatcher: path,
				}
				newRoute.addMethodHandler(method, middle)
				r.addChildren(newRoute)
			}
		} else if l < pathLength {
			path = path[l:]
			c := findRouteByLabel(r, path[0])
			if c != nil {
				r = c
				continue
			}
			newRoute := &route{
				label:        path[0],
				prefix:       path,
				regexMatcher: path,
			}
			newRoute.addMethodHandler(method, middle)
			r.addChildren(newRoute)
		} else {
			r.addMethodHandler(method, middle)
		}
		return
	}
}

func findRouteByLabel(r *route, label byte) *route {
	for i, value := range r.children {
		if value.label == label {
			return r.children[i]
		}
	}
	return nil
}

func convertToNew(prefix string, children childrenRoute, h *handler) *route {
	if h == nil {
		h = new(handler)
	}
	router := route{
		label:        prefix[0],
		prefix:       prefix,
		children:     children,
		handler:      h,
		regexMatcher: prefix,
	}
	return &router
}

func (r *route) addChildren(c *route) {
	r.children = append(r.children, c)
}

func (r *route) getHandlerByMethod(method string) iType.BindMiddle {
	switch method {
	case GET:
		return r.handler.get
	case POST:
		return r.handler.post
	case PUT:
		return r.handler.put
	case DELETE:
		return r.handler.delete
	}
	return nil
}

func (r *route) addMethodHandler(method string, m iType.ExtendMiddleSlice) {
	bm := iType.CombineMiddle(m)
	if r.handler == nil {
		r.handler = new(handler)
	}
	switch method {
	case POST:
		r.handler.post = bm
	case PUT:
		r.handler.put = bm
	case GET:
		r.handler.get = bm
	case DELETE:
		r.handler.delete = bm
	}
}
