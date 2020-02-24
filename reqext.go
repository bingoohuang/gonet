package gonet

import "log"

// HTTPGet 表示一次HTTP的Get调用
func HTTPGet(url string) ([]byte, error) {
	req, err := NewReqOption().Get(url)
	if err != nil {
		return nil, err
	}

	return req.Bytes()
}

// RestGet 发起一次HTTP GET调用，并且反序列化JSON到v代表的指针中。
func (s *ReqOption) RestGet(url string, v interface{}) error {
	req, err := s.Get(url)
	if err != nil {
		return err
	}

	return req.ToJSON(v)
}

// RestGet 发起一次HTTP GET调用，并且反序列化JSON到v代表的指针中。
func RestGet(url string, v interface{}) error {
	return NewReqOption().RestGet(url, v)
}

// RestPost 表示一次HTTP的POST调用
func RestPost(url string, req interface{}, rsp interface{}) ([]byte, error) {
	return NewReqOption().RestPostFn(url, req, rsp, nil)
}

// RestPostFn ...
func (s *ReqOption) RestPostFn(url string, req interface{}, rsp interface{}, fn func(*HTTPReq)) ([]byte, error) {
	resp, err := s.Post(url)
	if err != nil {
		return nil, err
	}

	if fn != nil {
		fn(resp)
	}

	if err = resp.JSONBody(req); err != nil {
		return nil, err
	}

	if rsp != nil {
		return nil, resp.ToJSON(rsp)
	}

	return resp.Bytes()
}

// Get returns *HTTPReq with GET method.
func (s *ReqOption) Get(url string) (*HTTPReq, error) {
	return s.Req(url, "GET")
}

// Get returns *HTTPReq with GET method.
func Get(url string) (*HTTPReq, error) {
	return NewReqOption().Get(url)
}

// MustGet  returns *HTTPReq with GET method.
func MustGet(url string) *HTTPReq {
	return NewReqOption().MustGet(url)
}

// MustGet  returns *HTTPReq with GET method.
func (s *ReqOption) MustGet(url string) *HTTPReq {
	req, err := s.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	return req
}

// Post returns *HTTPReq with POST method.
func Post(url string) (*HTTPReq, error) {
	return NewReqOption().Post(url)
}

// Post returns *HTTPReq with POST method.
func (s *ReqOption) Post(url string) (*HTTPReq, error) {
	return s.Req(url, "POST")
}

// MustPost returns *HTTPReq with POST method.
func MustPost(url string) *HTTPReq {
	return NewReqOption().MustPost(url)
}

// MustPost returns *HTTPReq with POST method.
func (s *ReqOption) MustPost(url string) *HTTPReq {
	req, err := s.Post(url)
	if err != nil {
		log.Fatal(err)
	}

	return req
}

// Put returns *HTTPReq with PUT method.
func Put(url string) (*HTTPReq, error) {
	return NewReqOption().Put(url)
}

// Put returns *HTTPReq with PUT method.
func (s *ReqOption) Put(url string) (*HTTPReq, error) {
	return s.Req(url, "PUT")
}

// MustPut returns *HTTPReq with PUT method.
func MustPut(url string) *HTTPReq {
	return NewReqOption().MustPut(url)
}

// MustPut returns *HTTPReq with PUT method.
func (s *ReqOption) MustPut(url string) *HTTPReq {
	req, err := s.Put(url)
	if err != nil {
		log.Fatal(err)
	}

	return req
}

// Delete returns *HTTPReq with DELETE method.
func Delete(url string) (*HTTPReq, error) {
	return NewReqOption().Delete(url)
}

// Delete returns *HTTPReq with DELETE method.
func (s *ReqOption) Delete(url string) (*HTTPReq, error) {
	return s.Req(url, "DELETE")
}

// MustDelete returns *HTTPReq with DELETE method.
func MustDelete(url string) *HTTPReq {
	return NewReqOption().MustDelete(url)
}

// MustDelete returns *HTTPReq with DELETE method.
func (s *ReqOption) MustDelete(url string) *HTTPReq {
	req, err := s.Delete(url)
	if err != nil {
		log.Fatal(err)
	}

	return req
}

// Head returns *HTTPReq with HEAD method.
func Head(url string) (*HTTPReq, error) {
	return NewReqOption().Head(url)
}

// Head returns *HTTPReq with HEAD method.
func (s *ReqOption) Head(url string) (*HTTPReq, error) {
	return s.Req(url, "HEAD")
}

// MustHead returns *HTTPReq with Head method.
func MustHead(url string) *HTTPReq {
	return NewReqOption().MustHead(url)
}

// MustHead returns *HTTPReq with Head method.
func (s *ReqOption) MustHead(url string) *HTTPReq {
	req, err := s.Head(url)
	if err != nil {
		log.Fatal(err)
	}

	return req
}

// Patch returns *HTTPReq with PATCH method.
func Patch(url string) (*HTTPReq, error) {
	return NewReqOption().Patch(url)
}

// Patch returns *HTTPReq with PATCH method.
func (s *ReqOption) Patch(url string) (*HTTPReq, error) {
	return s.Req(url, "PATCH")
}

// MustPatch returns *HTTPReq with Patch method.
func MustPatch(url string) *HTTPReq {
	return NewReqOption().MustPatch(url)
}

// MustPatch returns *HTTPReq with Patch method.
func (s *ReqOption) MustPatch(url string) *HTTPReq {
	req, err := s.Patch(url)
	if err != nil {
		log.Fatal(err)
	}

	return req
}
