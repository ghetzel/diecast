package internal

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	_ "github.com/rclone/rclone/backend/all"
	rclone_fs "github.com/rclone/rclone/fs"
	rclone_config "github.com/rclone/rclone/fs/config"
	rclone_configfile "github.com/rclone/rclone/fs/config/configfile"
)

var rcloneInstances = make(map[string]rclone_fs.Fs)

type rcloneOpenerFunc func(*rcloneFsFile) (io.ReadCloser, error)

type RequiredValue bool

func (self RequiredValue) String() string {
	return `{REQUIRED}`
}

type RcloneFS struct {
	fs.FS
	Name    string
	Type    string
	Root    string
	Options map[string]interface{}
	vfs     rclone_fs.Fs
	rcctx   context.Context
}

func currentUser() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	} else {
		return ``
	}
}

var DefaultRemoteTypeOptions = map[string]map[string]interface{}{
	`s3`: {
		`provider`: `AWS`,
		`env_auth`: true,
	},
	`sftp`: {
		`user`:     currentUser(),
		`key_file`: fileutil.MustExpandUser(`~/.ssh/id_rsa`),
	},
	`b2`: {
		`fast-list`: true,
		`account`:   RequiredValue(true),
		`key`:       RequiredValue(true),
	},
}

func CreateRcloneFilesystem(name string, typeRootPair string, options map[string]interface{}) (*RcloneFS, error) {
	var fstype, rt = stringutil.SplitPair(typeRootPair, `:`)
	var rcfs = &RcloneFS{
		Name:    name,
		Type:    fstype,
		Root:    rt,
		Options: options,
	}

	if err := rcfs.validate(); err == nil {
		return rcfs, nil
	} else {
		return nil, err
	}
}

// Returns a merged set of options that considers type-specific overrides and
// all configuration sources.
func (self *RcloneFS) opts() map[string]interface{} {
	var options = make(map[string]interface{})

	if defaults, ok := DefaultRemoteTypeOptions[self.Type]; ok {
		for k, v := range defaults {
			options[k] = v
		}
	}

	for k, v := range self.Options {
		options[k] = v
	}

	return options
}

// Ensures that the RcloneFS is usable before attempting to work with it.
func (self *RcloneFS) validate() error {
	if self.Name == `` {
		return fmt.Errorf("must specify a media source name")
	}

	if self.Type == `` {
		return fmt.Errorf("no type specified")
	}

	if len(self.Options) == 0 {
		self.Options = make(map[string]interface{})
	}

	self.rcctx = context.Background()
	self.Root = fileutil.MustExpandUser(self.Root)

	if err := self.generateAndSetRcloneConfig(); err != nil {
		return err
	}

	// log.Debugf("vfs/rclone type=%v root=%v", self.Name, self.Type, fmt.Sprintf("%s:%s", self.Name, self.Root))

	if vfs, ok := rcloneInstances[self.Name]; ok {
		self.vfs = vfs
	} else if vfs, err := rclone_fs.NewFs(
		self.rcctx,
		fmt.Sprintf("%s:%s", self.Name, self.Root),
	); err == nil {
		self.vfs = vfs
		rcloneInstances[self.Name] = self.vfs
	} else {
		return err
	}

	if self.vfs == nil {
		return fmt.Errorf("vfs not available")
	} else {
		self.vfs.Features()
	}

	return nil
}

// Generate, write out, and make available the RClone configuration data.
func (self *RcloneFS) generateAndSetRcloneConfig() error {
	var path = self.workingConfigPath()
	var opts = make(map[string]interface{})

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	// for each option
	for kv := range maputil.M(self.opts()).Iter(maputil.IterOptions{
		SortKeys: true,
	}) {
		if r, ok := kv.Value.(RequiredValue); ok && bool(r) {
			return fmt.Errorf("option %q must be specified for driver type %s", kv.K, self.Type)
		}

		opts[kv.K] = kv.Value
	}

	// log.Debugf("[source=%q] writing RClone configuration to %v", self.Name, path)

	if _, err := fileutil.WriteFile(self.rcloneConfigString(opts), path); err != nil {
		return err
	}

	if err := rclone_config.SetConfigPath(path); err != nil {
		return err
	}

	rclone_configfile.Install()

	rclone_fs.LogOutput = func(level rclone_fs.LogLevel, text string) {
		for _, substring := range []string{
			`Can't follow symlink without`,
			`Can't transfer non file/directory`,
		} {
			if strings.Contains(text, substring) {
				return
			}
		}

		// log.Debugf("vfs/rclone: %v", text)
	}

	return nil
}

// Locate the RClone configuration for this media source.
func (self *RcloneFS) workingConfigPath() string {
	var path = fmt.Sprintf("~/.cache/diecast/fs-%s.conf", self.Name)

	return fileutil.MustExpandUser(path)
}

// Generate the RClone configuration for communicating with this media source.
func (self *RcloneFS) rcloneConfigString(opts map[string]interface{}) string {
	var lines = make([]string, 0)

	lines = append(lines, fmt.Sprintf("[%s]", self.Name))
	lines = append(lines, fmt.Sprintf("type = %s", self.Type))

	for k, v := range opts {
		lines = append(lines, fmt.Sprintf("%s = %v", k, v))
	}

	return strings.Join(lines, "\n") + "\n"
}

func (self *RcloneFS) remotePath(path string) string {
	return fmt.Sprintf("%v:%v", self.Name, path)
}

func (self *RcloneFS) Open(name string) (fs.File, error) {
	if err := self.validate(); err != nil {
		return nil, fmt.Errorf("invalid media source: %v", err)
	}

	var entries rclone_fs.DirEntries
	var dir, file = filepath.Split(name)
	dir = filepath.Clean(filepath.Join(self.Root, dir))

	var entry = newRcloneFsFile(self, dir, file)
	var done bool

	if obj, oerr := self.vfs.NewObject(self.rcctx, name); oerr == nil {
		entry.SetName(name)
		entry.SetIsDir(false)
		entry.SetSize(obj.Size())
		entry.SetFileOpener(obj.Size(), func(e *rcloneFsFile) (io.ReadCloser, error) {
			return obj.Open(self.rcctx)
		})

		return entry, nil
	} else if oerr == rclone_fs.ErrorIsDir {
		if e, err := self.vfs.List(self.rcctx, dir); err == nil {
			entries = e
		} else {
			return nil, err
		}
	} else {
		return nil, oerr
	}

	entries.ForDir(func(d rclone_fs.Directory) {
		var fileobj = newRcloneFsFile(self, d.String(), ``)
		entry.SetIndex(entry.Len())
		entry.AddChild(fileobj)
	})

	entries.ForObject(func(o rclone_fs.Object) {
		var dir, filename = filepath.Split(o.String())
		var childobj = newRcloneFsFile(self, dir, filename)

		if done {
			return
		} else if file != `` && o.String() == file {
			childobj.SetFileOpener(o.Size(), func(e *rcloneFsFile) (io.ReadCloser, error) {
				return o.Open(self.rcctx)
			})

			done = true
		} else {
			childobj.SetFileOpener(o.Size(), func(e *rcloneFsFile) (io.ReadCloser, error) {
				return o.Open(self.rcctx)
			})

			entry.SetIndex(entry.Len())
			entry.AddChild(childobj)

		}
	})

	return entry, nil
}

type rcloneFsFile struct {
	*fileutil.FileInfo
	size     int64
	opener   rcloneOpenerFunc
	dir      string
	filename string
	basefs   *RcloneFS
	nfo      *fileutil.FileInfo
	rc       io.ReadCloser
	children []*rcloneFsFile
	mimetype string
	index    int
	openlock sync.Mutex
}

func newRcloneFsFile(base *RcloneFS, dir string, name string) *rcloneFsFile {
	return &rcloneFsFile{
		FileInfo: fileutil.NewFileInfo(),
		basefs:   base,
		children: make([]*rcloneFsFile, 0),
		dir:      dir,
		filename: name,
		mimetype: fileutil.GetMimeType(name, ``),
	}
}

func (self *rcloneFsFile) SetFileOpener(size int64, opener rcloneOpenerFunc) {
	self.size = size
	self.opener = opener
}

func (self *rcloneFsFile) AddChild(child *rcloneFsFile) {
	self.children = append(self.children, child)
}

func (self *rcloneFsFile) engage() (io.ReadCloser, error) {
	self.openlock.Lock()
	defer self.openlock.Unlock()

	if self.rc == nil {
		if self.opener == nil {
			return nil, fmt.Errorf("no opener for media entry")
		}

		if rc, err := self.opener(self); err == nil {
			self.rc = rc
		} else {
			return nil, err
		}
	}

	return self.rc, nil
}

func (self *rcloneFsFile) Stat() (fs.FileInfo, error) {
	if self.nfo == nil {
		self.nfo = fileutil.NewFileInfo()

		if self.filename == `` {
			self.nfo.SetIsDir(true)
			self.nfo.SetName(self.dir)
		} else {
			self.nfo.SetIsDir(false)
			self.nfo.SetName(self.filename)
		}

		self.nfo.SetSize(self.size)
	}

	if self.nfo.Name() == `` {
		return nil, fmt.Errorf("invalid entry name")
	}

	return self.nfo, nil
}

func (self *rcloneFsFile) Name() string {
	return typeutil.OrString(self.filename, strings.TrimSuffix(self.dir, `/`)+`/`)
}

func (self *rcloneFsFile) IsDir() bool {
	return len(self.filename) == 0
}

func (self *rcloneFsFile) Type() fs.FileMode {
	return 0
}

func (self *rcloneFsFile) Info() (fs.FileInfo, error) {
	return self.Stat()
}

func (self *rcloneFsFile) Read(b []byte) (int, error) {
	if rc, err := self.engage(); err == nil {
		return rc.Read(b)
	} else {
		return 0, err
	}
}

func (self *rcloneFsFile) Close() error {
	if rc, err := self.engage(); err == nil {
		return rc.Close()
	} else {
		return err
	}
}

func (self *rcloneFsFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if self.IsDir() {
		if n < len(self.children) {
			self.sortChildren()
			var children = self.children

			if n >= 0 {
				children = children[0:n]
			} else {
				n = len(children)
			}

			var results = make([]fs.DirEntry, n)

			for i, child := range children {
				results[i] = child
			}

			return results, nil
		} else {
			return make([]fs.DirEntry, 0), io.EOF
		}
	}

	return nil, fmt.Errorf("not a directory")
}

func (self *rcloneFsFile) sortChildren() {
	if len(self.children) > 0 {
		sort.Slice(self.children, func(i int, j int) bool {
			var ic = self.children[i]
			var jc = self.children[j]
			return ic.sortKey() < jc.sortKey()
		})
	}
}

func (self *rcloneFsFile) sortKey() string {
	var epoch string
	var name string = self.Name()

	if self.IsDir() {
		epoch = `0`
	} else {
		epoch = `1`
	}

	return epoch + `:` + strings.ToLower(name)
}
func (self *rcloneFsFile) Len() int {
	return len(self.children)
}

func (self *rcloneFsFile) Index() int {
	return self.index
}

func (self *rcloneFsFile) SetIndex(i int) {
	self.index = i
}
