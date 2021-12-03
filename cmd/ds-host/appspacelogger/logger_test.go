package appspacelogger

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestGetChunkEmptyFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	logFile := filepath.Join(dir, "log.txt")
	logStr := ""
	err = os.WriteFile(logFile, []byte(logStr), 0664)
	if err != nil {
		t.Fatal(err)
	}

	fd, err := os.OpenFile(logFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0664)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		from     int64
		to       int64
		expected domain.LogChunk
	}{
		{from: 0, to: 0, expected: domain.LogChunk{
			From: 0, To: 0, Content: ""}},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%v - %v", c.from, c.to), func(t *testing.T) {
			out, err := getChunk(fd, c.from, c.to)
			if err != nil {
				t.Fatal(err)
			}
			if !cmp.Equal(out, c.expected) {
				t.Error(cmp.Diff(out, c.expected))
			}
		})
	}
}

func TestGetChunk(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	logFile := filepath.Join(dir, "log.txt")
	logStr := "abcd\nefgh\nijkl\n"
	err = os.WriteFile(logFile, []byte(logStr), 0664)
	if err != nil {
		t.Fatal(err)
	}

	fd, err := os.OpenFile(logFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0664)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		from     int64
		to       int64
		expected domain.LogChunk
	}{
		{from: 0, to: 0, expected: domain.LogChunk{
			From: 0, To: 15, Content: "abcd\nefgh\nijkl\n"}},
		{from: 2, to: 0, expected: domain.LogChunk{
			From: 5, To: 15, Content: "efgh\nijkl\n"}},
		{from: -11, to: 0, expected: domain.LogChunk{
			From: 5, To: 15, Content: "efgh\nijkl\n"}},
		{from: -20, to: 0, expected: domain.LogChunk{
			From: 0, To: 15, Content: "abcd\nefgh\nijkl\n"}},
		{from: 2, to: 12, expected: domain.LogChunk{
			From: 5, To: 10, Content: "efgh\n"}},
		{from: 2, to: 15, expected: domain.LogChunk{
			From: 5, To: 15, Content: "efgh\nijkl\n"}},
		{from: 0, to: 2, expected: domain.LogChunk{
			From: 0, To: 0, Content: ""}},
		{from: 7, to: 8, expected: domain.LogChunk{
			From: 0, To: 0, Content: ""}},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%v - %v", c.from, c.to), func(t *testing.T) {
			out, err := getChunk(fd, c.from, c.to)
			if err != nil {
				t.Fatal(err)
			}
			if !cmp.Equal(out, c.expected) {
				t.Error(cmp.Diff(out, c.expected))
			}
		})
	}
}

func TestGetLast(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	logFile := filepath.Join(dir, "log.txt")
	logStr := "abcd\nefgh\nijkl\n"
	err = os.WriteFile(logFile, []byte(logStr), 0664)
	if err != nil {
		t.Fatal(err)
	}

	l := Logger{
		logPath: logFile}
	l.open()

	chunk, err := l.GetLastBytes(11)
	if err != nil {
		t.Fatal(err)
	}
	expected := domain.LogChunk{
		From: 5, To: 15, Content: "efgh\nijkl\n"}
	if !cmp.Equal(chunk, expected) {
		t.Error(cmp.Diff(chunk, expected))
	}
}
