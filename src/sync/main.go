package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	errors2 "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	flagRegistry string
	flagSrc      string
	flagDst      string
	flagDownload bool
	flagSync     bool
	flagPull     bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:  "app",
		Long: `sync image`,
		Run: func(cmd *cobra.Command, args []string) {
			log.SetLevel(log.DebugLevel)
			log.SetReportCaller(true)
			err := run()
			if err != nil {
				log.Debugf("err: %+v", err)
				return
			}
		},
	}
	rootCmd.Flags().StringVar(&flagRegistry, "registry", "registry.cn-huhehaote.aliyuncs.com/feng-566/", "target registry")
	rootCmd.Flags().StringVar(&flagSrc, "src", "./src", "src dir")
	rootCmd.Flags().StringVar(&flagDst, "dst", "./dst", "dst dir")
	rootCmd.Flags().BoolVar(&flagDownload, "download", false, "download yaml")
	rootCmd.Flags().BoolVar(&flagSync, "sync", true, "sync image")
	rootCmd.Flags().BoolVar(&flagPull, "pull", true, "set to false push local")

	rootCmd.Execute()

}

func run() error {
	var err error
	if flagDownload {
		err = download()
		if err != nil {
			return errors2.WithStack(err)
		}
	}

	syncer := NewSyncer(flagRegistry)
	err = syncer.syncImage(flagSrc, flagDst)
	if err != nil {
		return errors2.WithStack(err)
	}
	err = syncer.log()
	if err != nil {
		return errors2.WithStack(err)
	}

	if flagSync {
		err = syncer.sync()
		if err != nil {
			return errors2.WithStack(err)
		}
	}

	return nil
}

type Syncer struct {
	Registry string
	mapping  [][2]string
}

func NewSyncer(registry string) *Syncer {
	return &Syncer{
		Registry: registry,
	}
}

func (s *Syncer) syncImage(src, dst string) error {
	fs, err := ioutil.ReadDir(src)
	if err != nil {
		return errors2.WithStack(err)
	}

	err = os.MkdirAll(dst, 0777)
	if err != nil {
		return errors2.WithStack(err)
	}

	for _, v := range fs {
		if v.IsDir() {
			continue
		}

		err := func() error {
			srcFd, err := os.Open(filepath.Join(src, v.Name()))
			if err != nil {
				return errors2.WithStack(err)
			}
			defer srcFd.Close()

			dstFd, err := os.OpenFile(filepath.Join(dst, v.Name()), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
			if err != nil {
				return errors2.WithStack(err)
			}
			defer dstFd.Close()

			rd := bufio.NewReader(srcFd)
			for {
				srcLine, err := rd.ReadBytes('\n')
				if err != nil {
					if err == io.EOF {
						_, err = rd.WriteTo(dstFd)
						if err != nil {
							return errors2.WithStack(err)
						}
						break
					}
					return errors2.WithStack(err)
				}

				dstLine, err := s.replaceImage(srcLine)
				if err != nil {
					return errors2.WithStack(err)
				}

				_, err = dstFd.Write(dstLine)
				if err != nil {
					return errors2.WithStack(err)
				}
			}
			return nil
		}()

		if err != nil {
			return errors2.WithStack(err)
		}
	}

	return nil
}

var (
	expImageFull = regexp.MustCompile(`(?:image:|-image",)\s(?:'|")?([^@]+)(@sha256:\w+)?(?:'|")?`)

	expImageName = regexp.MustCompile(`(?:.+/)*(.+)$`)

	errNotMatch = errors2.New("not match")
)

func (s *Syncer) replaceImage(line []byte) ([]byte, error) {
	src, dst, err := s.imageMapping(line)
	if err != nil {
		if err == errNotMatch {
			return line, nil
		}
		return nil, errors2.WithStack(err)
	}

	s.mapping = append(s.mapping, [2]string{src, dst})

	dstLine := strings.Replace(string(line), src, dst, 1)
	return []byte(dstLine), nil
}

func (s *Syncer) imageMapping(line []byte) (string, string, error) {
	if !bytes.Contains(line, []byte("image")) || !expImageFull.Match(line) {
		return "", "", errNotMatch
	}

	groups := expImageFull.FindSubmatch(line)
	if len(groups) == 0 {
		log.Debugf("line: %s", line)
	}

	srcTag := groups[1]
	sha := groups[2]

	groups2 := expImageName.FindSubmatch(srcTag)
	lastName := groups2[1]

	var prefix string
	if bytes.Contains(srcTag, []byte("trigger")) {
		prefix = "trigger-"
	} else {
		prefix = "tekton-"
	}
	dstTag := s.Registry + prefix + string(lastName)
	return string(srcTag) + string(sha), dstTag, nil
}

func (s *Syncer) log() error {
	logFd, err := os.OpenFile("./sync.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return errors2.WithStack(err)
	}

	for _, v := range s.mapping {
		_, err := logFd.WriteString(fmt.Sprintf("%s => %s\n", v[0], v[1]))
		if err != nil {
			return errors2.WithStack(err)
		}
	}
	return nil
}

func (s *Syncer) sync() error {
	for _, v := range s.mapping {
		err := pullAndPush(v[0], v[1])
		if err != nil {
			return errors2.WithStack(err)
		}
	}
	return nil
}

func pullAndPush(src, dst string) error {
	log.Debugf("sync image: %s => %s", src, dst)

	var err error
	if flagPull {
		err = runCmd("docker", "pull", src)
		if err != nil {
			return errors2.WithStack(err)
		}
	}

	err = runCmd("docker", "tag", src, dst)
	if err != nil {
		return errors2.WithStack(err)
	}

	err = runCmd("docker", "push", dst)
	if err != nil {
		return errors2.WithStack(err)
	}

	return nil
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return errors2.WithMessagef(err, "stdout: %s, stderr: %s", stdout.String(), stderr.String())
	}
	return nil
}

func download() error {
	return runCmd("sh", "./download.sh")
}
