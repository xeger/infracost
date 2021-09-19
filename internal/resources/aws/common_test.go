package aws_test

import (
	"bytes"
	"context"
	"fmt"
	"hash/crc32"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	usage "github.com/infracost/infracost/internal/usage/aws"
)

type stubbedRequest struct {
	bodyFragments  []string
	response       string
	responseStatus int
}

func (sr *stubbedRequest) Then(status int, response string) {
	sr.responseStatus = status
	sr.response = response
}

type stubbedAWS struct {
	t        *testing.T
	server   *httptest.Server
	ctx      context.Context
	requests []*stubbedRequest
	usage    map[string]interface{}
}

func (sa *stubbedAWS) expectSuccess(op string, err error) {
	if err != nil {
		sa.t.Fatalf("Expected %s to succeed, got %s", op, err)
	}
}

func (sa *stubbedAWS) expectUsage(key string, expected interface{}) {
	actual := sa.usage[key]
	if actual != expected {
		sa.t.Fatalf("Expected %s %v %T, got %v %T", key, expected, expected, actual, actual)
	}
}

func (sa *stubbedAWS) writeResponse(w http.ResponseWriter, status int, body string) {
	hash := crc32.NewIEEE()
	hash.Write([]byte(body))
	crc32 := hash.Sum32()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Amz-Crc32", fmt.Sprintf("%d", crc32))
	w.WriteHeader(status)

	_, err := w.Write([]byte(body))
	if err != nil {
		sa.t.Fatalf("Cannot write stubbed HTTP response: %s", err)
	}
}

func (sa *stubbedAWS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(r.Body)
	r.Body.Close()
	body := buf.String()

	for _, sr := range sa.requests {
		match := true
		for _, fragment := range sr.bodyFragments {
			match = match && strings.Contains(body, fragment)
		}
		if match {
			sa.writeResponse(w, sr.responseStatus, sr.response)
			return
		}
	}
	sa.t.Fatalf("received unexpected stubbed AWS call: %s %s %s", r.Method, r.URL, body)
}

func (sa *stubbedAWS) WhenBody(fragments ...string) *stubbedRequest {
	sr := &stubbedRequest{
		bodyFragments: fragments,
	}
	sa.requests = append(sa.requests, sr)
	return sr
}

func (sa *stubbedAWS) Close() {
	sa.server.Close()
}

func stubAWS(t *testing.T) *stubbedAWS {
	stub := &stubbedAWS{
		t:        t,
		requests: make([]*stubbedRequest, 0),
		usage:    make(map[string]interface{}),
	}
	stub.server = httptest.NewServer(stub)
	stub.ctx = usage.WithTestEndpoint(context.TODO(), stub.server.URL)
	return stub
}
