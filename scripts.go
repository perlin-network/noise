//go:generate go run scripts.go

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var (
	goPath = os.Getenv("GOPATH")
)

func main() {
	if err := generateProtos("."); err != nil {
		fmt.Printf("%+v", err)
	}
}

func generateProtos(dir string) error {

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		// skip vendor directory
		if info.IsDir() && info.Name() == "vendor" {
			return filepath.SkipDir
		}
		// find all protobuf files
		if filepath.Ext(path) == ".proto" {
			// args
			args := []string{
				"-I=.",
				fmt.Sprintf("-I=%s/src", goPath),
				fmt.Sprintf("-I=%s/src/github.com/gogo/protobuf/protobuf", goPath),
				fmt.Sprintf("--proto_path=%s/src/github.com", goPath),
				"--gogofaster_out=Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types:.",
				path,
			}
			cmd := exec.Command("protoc", args...)
			err = cmd.Run()
			if err != nil {
				return err
			}
		}
		return nil
	})
}
