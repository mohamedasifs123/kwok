/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package file

import (
	"context"
	"io/fs"
	"testing"
)

func TestDownloadWithCacheAndExtract(t *testing.T) {
	type args struct {
		ctx      context.Context
		cacheDir string
		src      string
		dest     string
		match    string
		mode     fs.FileMode
		quiet    bool
		clean    bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DownloadWithCacheAndExtract(tt.args.ctx, tt.args.cacheDir, tt.args.src, tt.args.dest, tt.args.match, tt.args.mode, tt.args.quiet, tt.args.clean); (err != nil) != tt.wantErr {
				t.Errorf("DownloadWithCacheAndExtract() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDownloadWithCache(t *testing.T) {
	type args struct {
		ctx      context.Context
		cacheDir string
		src      string
		dest     string
		mode     fs.FileMode
		quiet    bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DownloadWithCache(tt.args.ctx, tt.args.cacheDir, tt.args.src, tt.args.dest, tt.args.mode, tt.args.quiet); (err != nil) != tt.wantErr {
				t.Errorf("DownloadWithCache() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getCachePath(t *testing.T) {
	type args struct {
		cacheDir string
		src      string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getCachePath(tt.args.cacheDir, tt.args.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("getCachePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getCachePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getCacheOrDownload(t *testing.T) {
	type args struct {
		ctx      context.Context
		cacheDir string
		src      string
		mode     fs.FileMode
		quiet    bool
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getCacheOrDownload(tt.args.ctx, tt.args.cacheDir, tt.args.src, tt.args.mode, tt.args.quiet)
			if (err != nil) != tt.wantErr {
				t.Errorf("getCacheOrDownload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getCacheOrDownload() = %v, want %v", got, tt.want)
			}
		})
	}
}
