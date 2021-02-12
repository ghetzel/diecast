package diecast

import (
	"net/http"
	"os"
	"testing"

	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/testify/require"
)

func TestFancyMapJoin(t *testing.T) {
	var assert = require.New(t)

	assert.Equal(`hello=there`, fancyMapJoin(map[string]interface{}{
		`hello`: `there`,
	}))

	assert.Equal(`hello=there&how=are you?`, fancyMapJoin(map[string]interface{}{
		`hello`: `there`,
		`how`:   `are you?`,
	}))

	assert.Equal(`hello: there; how: are you?`, fancyMapJoin(map[string]interface{}{
		`_kvjoin`: `: `,
		`_join`:   `; `,
		`hello`:   `there`,
		`how`:     `are you?`,
	}))

	assert.Equal(`hello: "there"; how: "are you?"`, fancyMapJoin(map[string]interface{}{
		`_kvjoin`:  `: `,
		`_join`:    `; `,
		`_vformat`: "%q",
		`hello`:    `there`,
		`how`:      `are you?`,
	}))
}

func TestZipFS(t *testing.T) {
	var file http.File
	var stat os.FileInfo
	var fs, err = newZipFsFromFile(`./tests/zip-fs-test.zip`)

	require.NoError(t, err)
	require.NotNil(t, fs)

	file, err = fs.Open(`/`)
	require.NoError(t, err)

	stat, err = file.Stat()
	require.NoError(t, err)
	require.True(t, stat.IsDir())

	file, err = fs.Open(`README.md`)

	require.NoError(t, err)
	require.Contains(t, fileutil.Cat(file), `ZIPFS TEST`)

	// ===================================================================================================================
	var entries []string

	for _, entry := range fs.entries(`/`) {
		entries = append(entries, entry.Name())
	}

	require.Equal(t, []string{
		"subdir",
		"README.md",
	}, entries)

	// ===================================================================================================================
	entries = nil

	for _, entry := range fs.entries(`subdir`) {
		entries = append(entries, entry.Name())
	}

	require.Equal(t, []string{
		"more",
		"third.txt",
		"first.txt",
		"second.txt",
	}, entries)

	// ===================================================================================================================
	entries = nil

	for _, entry := range fs.entries(`subdir/more`) {
		entries = append(entries, entry.Name())
	}

	require.Equal(t, []string{
		"even-more",
		"fourth.txt",
		"fifth.txt",
	}, entries)
}
