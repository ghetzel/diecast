package internal

var rcloneInstances = make(map[string]rclone_fs.Fs)

type RequiredValue bool

func (self RequiredValue) String() string {
	return `{REQUIRED}`
}

type RcloneFS struct {
	fs.FS
	Name          string                 
	Type          string                 
	Root          string                 
	Options       map[string]interface{} 
	vfs           rclone_fs.Fs
	rcctx         context.Context
}

func currentUser() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	} else {
		return ``
	}
}

var DefaultRemoteTypeOptions = map[string]map[string]interface{}{
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
	var source = &RcloneFS{
		Name:    name,
		Type:    fstype,
		Root:    rt,
		Options: options,
	}

	if err := source.validate(); err == nil {
		return source, nil
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

	if self.Title == `` {
		self.Title = strings.Title(self.Name)
	}

	if len(self.Options) == 0 {
		self.Options = make(map[string]interface{})
	}

	self.rcctx = context.Background()
	self.Root = fileutil.MustExpandUser(self.Root)

	if err := self.generateAndSetRcloneConfig(); err != nil {
		return err
	}

	// log.Debugf("[source=%q] rclone type=%v root=%v", self.Name, self.Type, fmt.Sprintf("%s:%s", self.Name, self.Root))

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

	rclone_fs.LogPrint = func(level rclone_fs.LogLevel, text string) {
		for _, substring := range []string{
			`Can't follow symlink without`,
			`Can't transfer non file/directory`,
		} {
			if strings.Contains(text, substring) {
				return
			}
		}

		log.Debugf("[rclone] %v", text)
	}

	return nil
}

// Locate the RClone configuration for this media source.
func (self *RcloneFS) workingConfigPath() string {
	var path = fmt.Sprintf("~/.config/airstream/run/rclone-%s.conf", self.Name)

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

func (self *RcloneFS) Open(name string) (fs.File, error) {
	return self.OpenMedia(name, false)
}

func (self *RcloneFS) OpenMedia(name string, loadmeta bool) (*MediaEntry, error) {
	if err := self.validate(); err != nil {
		return nil, fmt.Errorf("invalid media source: %v", err)
	}

	var entries rclone_fs.DirEntries
	var dir, file = filepath.Split(name)
	var mediaEntry = NewMediaEntry(self, dir, file)
	var done bool

	if e, err := self.vfs.List(self.rcctx, name); err == nil {
		entries = e
	} else if log.ErrContains(err, `is a file not a directory`) {
		if e, err := self.vfs.List(self.rcctx, dir); err == nil {
			entries = e
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}

	entries.ForDir(func(d rclone_fs.Directory) {
		var entry = NewMediaEntry(self, d.String(), ``)
		entry.SetIndex(mediaEntry.Len())
		mediaEntry.AddChild(entry)
	})

	entries.ForObject(func(o rclone_fs.Object) {
		var dir, filename = filepath.Split(o.String())
		var entry = NewMediaEntry(self, dir, filename)

		if done {
			return
		} else if file != `` && o.String() == file {
			mediaEntry.SetFileOpener(o.Size(), func(e *MediaEntry) (io.ReadCloser, error) {
				return o.Open(self.rcctx)
			})

			if loadmeta {
				defer mediaEntry.LoadMetadata()
			}

			done = true
		} else {
			entry.SetFileOpener(o.Size(), func(e *MediaEntry) (io.ReadCloser, error) {
				return o.Open(self.rcctx)
			})

			if loadmeta {
				defer entry.LoadMetadata()
			}

			entry.SetIndex(mediaEntry.Len())

			mediaEntry.AddChild(entry)

		}
	})

	return mediaEntry, nil
}
