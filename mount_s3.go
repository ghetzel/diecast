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
)

type awsLog struct{}

func (self *awsLog) Log(data ...interface{}) {
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
//
type S3Mount struct {
	MountPoint string `json:"mount"`
	Path       string `json:"source"`
}

func (self *S3Mount) GetMountPoint() string {
	return self.MountPoint
}

func (self *S3Mount) GetTarget() string {
	return self.Path
}

func (self *S3Mount) WillRespondTo(name string, req *http.Request, requestBody io.Reader) bool {
	return true
}

func (self *S3Mount) OpenWithType(name string, req *http.Request, requestBody io.Reader) (*MountResponse, error) {
	if hf, err := self.Open(name); err == nil {
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

func (self *S3Mount) String() string {
	return fmt.Sprintf("%T('%s')", self, self.GetMountPoint())
}

func (self *S3Mount) Open(name string) (http.File, error) {
	if awsSession != nil {
		var remoteName = filepath.Join(self.Path, name)
		var bucket, key = stringutil.SplitPair(strings.TrimPrefix(remoteName, `/`), `/`)

		if obj, err := s3client().GetObject(&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		}); err == nil {
			var mr = NewMountResponse(remoteName, *obj.ContentLength, obj.Body)

			mr.ContentType = *obj.ContentType

			return mr, nil
		} else if log.ErrContains(err, `NoSuchKey`) {
			var fauxParent = newHttpFile(name, nil)

			fauxParent.SetIsDir(true)
			log.Noticef("s3: parent %v", fauxParent.Name())

			if err := s3client().ListObjectsPages(&s3.ListObjectsInput{
				Bucket: aws.String(bucket),
				Prefix: aws.String(strings.TrimSuffix(key, `/`) + `/`),
			}, func(page *s3.ListObjectsOutput, lastPage bool) bool {
				var subdirs = make(map[string]bool)

				for _, obj := range page.Contents {
					var lazyFile = newHttpFile(*obj.Key, nil)
					var objdir = filepath.Dir(*obj.Key)
					var add bool

					if *obj.Key == objdir {
						continue
					}

					lazyFile.SetFileSystem(self)
					lazyFile.SetName(*obj.Key)
					lazyFile.SetSize(*obj.Size)

					if objdir == key {
						if strings.HasPrefix(*obj.Key, objdir+`/`) {
							if _, ok := subdirs[objdir]; !ok {
								add = true
								lazyFile.SetIsDir(true)
								subdirs[objdir] = true
								log.Noticef("s3: dir %v", objdir)
							}
						} else {
							log.Noticef("s3: file %v", obj.Key)
						}
					}

					if add {
						var mr = NewMountResponse(*obj.Key, *obj.Size, lazyFile)
						fauxParent.AddChildFile(mr)
						log.Noticef("s3: add fcx %v", lazyFile.Name())
					}
				}

				return true
			}); err == nil {
				return fauxParent, nil
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

func existsInS3(bucket string, key string) (bool, error) {
	var client = s3client()

	if key == `` {
		if _, err := client.HeadBucket(&s3.HeadBucketInput{
			Bucket: aws.String(bucket),
		}); err == nil {
			return true, nil
		} else if log.ErrContains(err, `NoSuch`) || log.ErrContains(err, `NotFound`) {
			return false, nil
		} else {
			return false, err
		}
	} else {
		if _, err := client.HeadObject(&s3.HeadObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		}); err == nil {
			return true, nil
		} else if log.ErrContains(err, `NoSuch`) || log.ErrContains(err, `NotFound`) {
			return false, nil
		} else {
			return false, err
		}
	}
}
