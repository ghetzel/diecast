package diecast

import (
	"io"
	"net/http"
	"os"
	pathpkg "path"
	"path/filepath"
	"sort"
	"sync"

	"github.com/ghetzel/go-stockutil/log"
)

type httpFsWalkFilesFunc func(path string, info os.FileInfo, rs io.ReadSeeker, err error) error

func httpFsReadDir(fs http.FileSystem, name string) ([]os.FileInfo, error) {
	if file, err := fs.Open(name); err == nil {
		defer file.Close()
		return file.Readdir(0)
	} else {
		return nil, err
	}
}

func httpFsStat(fs http.FileSystem, name string) (os.FileInfo, error) {
	if file, err := fs.Open(name); err == nil {
		defer file.Close()
		return file.Stat()
	} else {
		return nil, err
	}
}

func httpFsWalk(fs http.FileSystem, root string, globPattern string, walkFn filepath.WalkFunc) error {
	if info, err := httpFsStat(fs, root); err == nil {
		return walk(fs, globPattern, root, info, walkFn)
	} else {
		return walkFn(root, nil, err)
	}
}

func httpFsReadDirNames(fs http.FileSystem, dirname string) ([]string, error) {
	if stats, err := httpFsReadDir(fs, dirname); err == nil {
		var names = make([]string, len(stats))

		for i := range stats {
			names[i] = stats[i].Name()
		}

		sort.Strings(names)
		return names, nil
	} else {
		return nil, err
	}
}

func walk(fs http.FileSystem, pattern string, path string, info os.FileInfo, walkFn filepath.WalkFunc) error {
	if err := walkFn(path, info, nil); err != nil {
		if info.IsDir() && err == filepath.SkipDir {
			return nil
		} else {
			return err
		}
	}

	if !info.IsDir() {
		return nil
	}

	var names, err = httpFsReadDirNames(fs, path)

	if err != nil {
		return walkFn(path, info, err)
	}

	var errchan = make(chan error, len(names))
	var wg sync.WaitGroup

	for _, name := range names {
		if pattern != `` {
			if matched, err := filepath.Match(pattern, name); err == nil {
				// log.Noticef("glob: %q ~ %q -> %v", pattern, name, matched)
				if !matched {
					continue
				}
			} else {
				return err
			}
		}

		wg.Add(1)

		go func(w *sync.WaitGroup, p string, n string) {
			defer w.Done()

			var filename = pathpkg.Join(p, n)
			var fileInfo, err = httpFsStat(fs, filename)

			if err != nil {
				if err := walkFn(filename, fileInfo, err); err != nil && err != filepath.SkipDir {
					errchan <- err
					return
				}
			} else if err := walk(fs, pattern, filename, fileInfo, walkFn); err != nil {
				if !fileInfo.IsDir() || err != filepath.SkipDir {
					errchan <- err
					return
				}
			}
		}(&wg, path, name)
	}

	wg.Wait()
	close(errchan)

	var merr error

	for err := range errchan {
		merr = log.AppendError(merr, err)
	}

	return merr
}

func httpFsWalkFiles(fs http.FileSystem, root string, walkFn httpFsWalkFilesFunc) error {
	if file, info, err := httpFsOpenStat(fs, root); err == nil {
		return walkFiles(fs, root, info, file, walkFn)
	} else {
		return walkFn(root, nil, nil, err)
	}
}

func walkFiles(fs http.FileSystem, path string, info os.FileInfo, file http.File, walkFn httpFsWalkFilesFunc) error {
	var err = walkFn(path, info, file, nil)
	file.Close()

	if err != nil {
		if info.IsDir() && err == filepath.SkipDir {
			return nil
		}
		return err
	}

	if !info.IsDir() {
		return nil
	}

	names, err := httpFsReadDirNames(fs, path)

	if err != nil {
		return walkFn(path, info, nil, err)
	}

	for _, name := range names {
		var filename = pathpkg.Join(path, name)
		var file, fileInfo, err = httpFsOpenStat(fs, filename)

		if err != nil {
			if err := walkFn(filename, nil, nil, err); err != nil && err != filepath.SkipDir {
				return err
			}
		} else if err := walkFiles(fs, filename, fileInfo, file, walkFn); err != nil {
			if !fileInfo.IsDir() || err != filepath.SkipDir {
				return err
			}
		}
	}

	return nil
}

func httpFsOpenStat(fs http.FileSystem, name string) (http.File, os.FileInfo, error) {
	if file, err := fs.Open(name); err == nil {
		if info, err := file.Stat(); err == nil {
			return file, info, nil
		} else {
			file.Close()
			return nil, nil, err
		}
	} else {
		return nil, nil, err
	}
}
