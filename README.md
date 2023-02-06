# easystruct

### Install
```sh
go install github.com/crayoned/easystruct@latest
```

### Run
```sh
easystruct model.go
```

Generates new function `FromRequest` for all structs in specific file to mapping request into struct. 

Supported data source: 
- header - req.Header
- query - req.URL.Query()
- formData - req.FormValue

Supported types:
- all primitive numbers(ints, uints, floats)
- string, []byte, []rune
- bool


Struct fields have to contain tag description with name `es`. Tag template `data_source=data_name`. For example:
```go
type SearchRequest struct {
  User   string `es:"header=x-user-id"`
  Limit  int    `es:"query=limit"`
  Search string `es:"query=search"`
} 
```
It generates new function like this:
```go
func(sr *SearchRequest) FromRequest(r *http.Request) error {
	if sval := r.Header.Get("x-user-id"); sval != "" {
		sr.User = sval
	}
	if sval := r.URL.Query().Get("limit"); sval != "" {
		nval, err := strconv.Atoi(sval)
		if err != nil {
			return err
		}
		sr.Limit = int(nval)
	}
	if sval := r.URL.Query().Get("search"); sval != "" {
		sr.Search = sval
	}
	return nil
}
```

Then you can use it in your http handlers:
```go
func processSearch(w http.ResponseWriter, r *http.Request) {
	var payload SearchRequest
	if err := payload.FromRequest(r); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// do something with payload
}
```
