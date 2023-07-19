// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This web server program is the simplified version of shakesapp.
// The original version is available here:
// https://github.com/GoogleCloudPlatform/golang-samples/blob/029368aa73a3407adc893af52e1512ff60551ad8/profiler/shakesapp/shakesapp/server.go
package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"

	"cloud.google.com/go/profiler"
	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

const bucketName = "dataflow-samples"
const bucketPrefix = "shakespeare/"
const version = "1.0.0"

type Request struct {
	Query string
}

func NewRequest(r *http.Request) *Request {
	q := r.URL.Query()
	value := q.Get("q")
	if value == "" {
		value = "hello"
	}
	return &Request{
		Query: value,
	}
}

type Response struct {
	MatchCount int64
}

func (r *Response) format() string {
	return fmt.Sprintf(`{"version": "%v", "count": %v}`, version, r.MatchCount)
}

func main() {
	cfg := profiler.Config{
		Service:        "shakesapp",
		ServiceVersion: version,
		DebugLogging:   true,
	}
	if err := profiler.Start(cfg); err != nil {
		log.Fatalf("failed to start profiler: %v", err)
	}

	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := NewRequest(r)

	resp := &Response{}
	texts, err := readFiles(ctx, bucketName, bucketPrefix)
	if err != nil {
		io.WriteString(w, fmt.Sprintf("failed to fetch files: %v", err))
		return
	}
	for _, text := range texts {
		for _, line := range strings.Split(text, "\n") {
			line, query := strings.ToLower(line), strings.ToLower(req.Query)
			// TODO: Compiling and matching a regular expression on every request
			// might be too expensive? Consider optimizing.
			isMatch, err := regexp.MatchString(query, line)
			if err != nil {
				io.WriteString(w, fmt.Sprintf("failed for regexp match: %v", err))
			}
			if isMatch {
				resp.MatchCount++
			}
		}
	}
	io.WriteString(w, resp.format())
}

// readFiles reads the content of files within the specified bucket with the
// specified prefix path in parallel and returns their content. It fails if
// operations to find or read any of the files fails.
func readFiles(ctx context.Context, bucketName, prefix string) ([]string, error) {
	type resp struct {
		s   string
		err error
	}

	client, err := storage.NewClient(ctx, option.WithoutAuthentication())
	if err != nil {
		return []string{}, fmt.Errorf("failed to create storage client: %s", err)
	}
	defer client.Close()

	bucket := client.Bucket(bucketName)

	var paths []string
	it := bucket.Objects(ctx, &storage.Query{Prefix: bucketPrefix})
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return []string{}, fmt.Errorf("failed to iterate over files in %s starting with %s: %w", bucketName, prefix, err)
		}
		if attrs.Name != "" {
			paths = append(paths, attrs.Name)
		}
	}

	resps := make(chan resp)
	for _, path := range paths {
		go func(path string) {
			obj := bucket.Object(path)
			r, err := obj.NewReader(ctx)
			if err != nil {
				resps <- resp{"", err}
			}
			defer r.Close()
			data, err := ioutil.ReadAll(r)
			resps <- resp{string(data), err}
		}(path)
	}
	ret := make([]string, len(paths))
	for i := 0; i < len(paths); i++ {
		r := <-resps
		if r.err != nil {
			err = r.err
		}
		ret[i] = r.s
	}
	return ret, err
}
