package http

// #cgo CFLAGS: -Wno-unused-parameter -Wno-switch-bool
// #include<spin-http.h>
// #include<stdlib.h>
import "C"
import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"unsafe"
)

//export spin_http_handle_http_request
func handle_http_request(req *C.spin_http_request_t, res *C.spin_http_response_t) {
	println("DEBUG SDK: entered handle_http_request")
	var body []byte
	if req.body.is_some {
		body = C.GoBytes(unsafe.Pointer(req.body.val.ptr), C.int(req.body.val.len))
	}
	println("DEBUG SDK: before method")
	method := methods[req.method]

	println("DEBUG SDK: before headers")
	header := fromSpinHeaders(&req.headers)

	println("DEBUG SDK: before URL")
	url := header.Get(HeaderFullUrl)

	println("DEBUG SDK: before NewRequest")

	r, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		res.status = C.uint16_t(http.StatusInternalServerError)
		return
	}

	r.Header = header
	r.Host = r.Header.Get("Host")
	r.RequestURI = C.GoStringN(req.uri.ptr, C.int(req.uri.len))
	r.RemoteAddr = r.Header.Get(HeaderClientAddr)

	w := newResponse()

	println("DEBUG SDK: before handler")

	// call user function
	handler(w, r)

	res.status = C.uint16_t(w.status)
	if len(w.header) > 0 {
		res.headers = C.spin_http_option_headers_t{
			is_some: true,
			val:     toSpinHeaders(w.header),
		}
	} else {
		res.headers = C.spin_http_option_headers_t{is_some: false}
	}

	res.body, err = toSpinBody(w.w)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func toSpinHeaders(hm http.Header) C.spin_http_headers_t {
	var reqHeaders C.spin_http_headers_t

	headersLen := len(hm)

	if headersLen > 0 {
		reqHeaders.len = C.ulong(headersLen)
		var x C.spin_http_tuple2_string_string_t
		reqHeaders.ptr = (*C.spin_http_tuple2_string_string_t)(C.malloc(C.size_t(headersLen) * C.size_t(unsafe.Sizeof(x))))
		headers := unsafe.Slice(reqHeaders.ptr, headersLen)

		idx := 0
		for k, v := range hm {
			headers[idx] = newSpinHeader(k, v[0])
			idx++
		}
	}
	return reqHeaders
}

func toSpinBody(body io.Reader) (C.spin_http_option_body_t, error) {
	var spinBody C.spin_http_option_body_t
	spinBody.is_some = false

	if body == nil {
		return spinBody, nil
	}

	buf := new(bytes.Buffer)
	len, err := buf.ReadFrom(body)
	if err != nil {
		return spinBody, err
	}

	if len > 0 {
		bodyPtr := (*C.uint8_t)(C.malloc(C.size_t(len)))
		copy(unsafe.Slice(bodyPtr, len), buf.Bytes())

		spinBody.is_some = true
		spinBody.val = C.spin_http_body_t{
			ptr: bodyPtr,
			len: C.size_t(len),
		}
	}

	return spinBody, nil
}

// newSpinHeader creates a new spinHeader with the given key/value.
func newSpinHeader(k, v string) C.spin_http_tuple2_string_string_t {
	return C.spin_http_tuple2_string_string_t{
		f0: C.spin_http_string_t{ptr: C.CString(k), len: C.size_t(len(k))},
		f1: C.spin_http_string_t{ptr: C.CString(v), len: C.size_t(len(v))},
	}
}

var methods = [...]string{
	"GET",
	"POST",
	"PUT",
	"DELETE",
	"PATCH",
	"HEAD",
	"OPTIONS",
}

func fromSpinHeaders(hm *C.spin_http_headers_t) http.Header {
	headersLen := int(hm.len)
	println("DEBUG SDK: headers length =", headersLen)
	println("DEBUG SDK: headers pointer =", unsafe.Pointer(hm.ptr))
	headers := make(http.Header, headersLen)

	var headersArr *C.spin_http_tuple2_string_string_t = hm.ptr
	println("DEBUG SDK: before unsafe.Slice")
	headersSlice := unsafe.Slice(headersArr, headersLen)
	println("DEBUG SDK: after unsafe.Slice")

	for i := 0; i < headersLen; i++ {
		println("DEBUG SDK: reading header index =", i)
		tuple := headersSlice[i]
		println("DEBUG SDK: header tuple loaded")
		println("DEBUG SDK: before header key")
		k := C.GoStringN(tuple.f0.ptr, C.int(tuple.f0.len))
		println("DEBUG SDK: after header key")

		println("DEBUG SDK: before header value")
		v := C.GoStringN(tuple.f1.ptr, C.int(tuple.f1.len))
		println("DEBUG SDK: after header value")

		headers.Add(k, v)
	}

	return headers
}
