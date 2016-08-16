package response

import (
	"encoding/json"
	"net/http"
	"fmt"
	"bytes"
	"crypto/md5"
)

type Response struct {
	Data  interface{}
	Error string

	Next     string
	Previous string

	Success bool
	Code    int

	noWrite bool
}

func (r *Response) MarshalJSON() ([]byte, error) {
	data := make(map[string]interface{})

	data["success"]  = r.Success
	data["code"]  = r.Code

	if r.Error != "" {
		data["error"] = r.Error
	} else if r.Data != nil {
		data["data"] = r.Data

		if r.Next != "" || r.Previous != "" {
			pagination := make(map[string]string)

			if r.Next != "" {
				pagination["next"] = r.Next
			}

			if r.Previous != "" {
				pagination["prev"] = r.Previous
			}

			data["paging"] = pagination
		}
	}

	return json.Marshal(data)
}

func (r *Response) NoWrite() {
	r.noWrite = true
}

func (res *Response) Write(w http.ResponseWriter, r *http.Request) {
	if res.noWrite {
		return
	}

	if (res.Code == http.StatusNotFound && res.Error == "") || (res.Error == "" && res.Data == nil && !res.Success) {
		res.Code = http.StatusNotFound
		res.Error = "not found"
	} else if res.Code == 0 && !res.Success {
		res.Code = http.StatusBadRequest
	} else if res.Code == 0 {
		res.Code = 200
	}

	out, err := json.Marshal(res)
	if err != nil {
		panic(err)
	}

	if r.FormValue("pretty") == "1" {
		buf := bytes.Buffer{}
		if err := json.Indent(&buf, out, "", "    "); err != nil {
			panic(err)
		}

		out = buf.Bytes()
	}

	out = bytes.Replace(out, []byte("\\u003c"), []byte("<"), -1)
	out = bytes.Replace(out, []byte("\\u003e"), []byte(">"), -1)
	out = bytes.Replace(out, []byte("\\u0026"), []byte("&"), -1)

	if res.Code >= 200 && res.Code < 300 {
		etag := fmt.Sprintf("\"%x\"", md5.Sum(out))

		w.Header().Set("Etag", etag)

		if match := r.Header.Get("if-none-match"); match != "" {
			if match == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(res.Code)

	fmt.Fprintf(w, "%s", out)
}

func (res *Response) Panic(msg interface{}) {
	res.noWrite = true
	panic(msg)
}
