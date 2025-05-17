package diecast

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/ghetzel/go-stockutil/executil"
	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/jszwec/s3fs"
)

type awsLog struct{}

func (awslog *awsLog) Log(data ...any) {
	log.Debug(data...)
}

var awsSession = func() *session.Session {
	var cfg = &aws.Config{
		Credentials: credentials.NewChainCredentials([]credentials.Provider{
			&credentials.EnvProvider{},
			&credentials.SharedCredentialsProvider{
				Filename: fileutil.MustExpandUser(`~/.aws/credentials`),
				Profile:  executil.Env(`AWS_PROFILE`, `default`),
			},
		}),
	}

	if log.VeryDebugging() {
		cfg.WithLogLevel(aws.LogDebugWithHTTPBody)
		cfg.Logger = new(awsLog)
	}

	if sess, err := session.NewSession(cfg); err == nil {
		if ep := executil.Env(`AWS_ENDPOINT_URL`); ep != `` {
			sess.Config.Endpoint = aws.String(ep)
		}

		return sess
	} else {
		log.Warningf("aws: invalid credentials: %v", err)
		return nil
	}
}()

// A S3Mount exposes the contents of a given filesystem directory.
// As is tradition with AWS client software, this package recongnizes and will
// honor several environment variable values for specifying configuration details
// to the client.  These variables include:
//
// - `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` representing the credentials to authenticate with
// - `AWS_REGION` to specify the region name
// - `AWS_PROFILE` to specify the named profile to utilize when reading from ~/.aws/credentials and ~/.aws/config
// - `AWS_ENDPOINT_URL` to override the HTTPS endpoint to use, namely for pointing to S3-compatible services.
type S3Mount struct {
	MountPoint string `json:"mount"`
	Path       string `json:"source"`
	fs         map[string]http.FileSystem
}

func (mount *S3Mount) s3fs(bucket string) http.FileSystem {
	if mount.fs == nil {
		mount.fs = make(map[string]http.FileSystem)
	}

	if _, ok := mount.fs[bucket]; !ok {
		mount.fs[bucket] = http.FS(s3fs.New(s3client(), bucket))
	}

	return mount.fs[bucket]
}

func (mount *S3Mount) GetMountPoint() string {
	return mount.MountPoint
}

func (mount *S3Mount) GetTarget() string {
	return mount.Path
}

func (mount *S3Mount) WillRespondTo(name string, req *http.Request, requestBody io.Reader) bool {
	return true
}

func (mount *S3Mount) OpenWithType(name string, req *http.Request, requestBody io.Reader) (*MountResponse, error) {
	if hf, err := mount.Open(name); err == nil {
		if mr, ok := hf.(*MountResponse); ok {
			if mimetype, err := figureOutMimeType(name, hf); err == nil {
				mr.ContentType = mimetype
				return mr, nil
			} else {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("invalid response")
		}
	} else {
		return nil, err
	}
}

func (mount *S3Mount) String() string {
	return fmt.Sprintf("%T('%s')", mount, mount.GetMountPoint())
}

func (mount *S3Mount) Open(name string) (http.File, error) {
	if awsSession != nil {
		name = filepath.Join(mount.Path, name)

		var bucket, key = stringutil.SplitPair(strings.TrimPrefix(name, `/`), `/`)

		if fsFile, err := mount.s3fs(bucket).Open(key); err == nil {
			if info, err := fsFile.Stat(); err == nil {
				var mr = NewMountResponse(name, info.Size(), fsFile)

				mr.setUnderlyingFile(fsFile, info)

				if ct := fileutil.GetMimeType(info.Name()); ct != `` {
					mr.ContentType = ct
				}

				return mr, nil
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("no credentials")
	}
}

func s3client() *s3.S3 {
	return s3.New(awsSession)
}
