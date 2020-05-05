package goproxy

import (
	"archive/zip"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/xakep666/licensevalidator/pkg/validation"

	"gopkg.in/src-d/go-license-detector.v3/licensedb/filer"
)

var (
	// ErrNotDirectory returned by ReadDir if we try to read non-directory path
	ErrNotDirectory = fmt.Errorf("not a directory")

	// ErrNotFile returned by ReadFile if we try to read non-file path
	ErrNotFile = fmt.Errorf("not a file")
)

type zipNode struct {
	children map[string]*zipNode
	file     *zip.File
}

func (n *zipNode) isDir() bool {
	if n == nil {
		return false
	}
	return n.file == nil || n.file.FileInfo().IsDir()
}

func (n *zipNode) isFile() bool {
	if n == nil {
		return false
	}
	return n.file != nil && !n.file.FileInfo().IsDir()
}

// ZipFiler is an implementation of filesystem for src-d license scanner based on zip archive
type ZipFiler struct {
	*zip.Reader
	// Module archives contains project root inside path "<module name>@<module version>"
	// So for correct license recognition we should trim this prefix from all paths
	validation.Module

	onceInitTree sync.Once
	tree         *zipNode
}

func (zf *ZipFiler) ReadFile(path string) ([]byte, error) {
	zf.initTree()

	parts := strings.Split(path, "/")

	node := zf.tree
	for _, part := range parts {
		if part == "" {
			continue
		}
		node = node.children[part]
		if node == nil {
			return nil, &os.PathError{
				Op:   "open",
				Path: path,
				Err:  os.ErrNotExist,
			}
		}
	}

	if node.isDir() {
		return nil, &os.PathError{
			Op:   "open",
			Path: path,
			Err:  ErrNotFile,
		}
	}

	reader, err := node.file.Open()
	if err != nil {
		return nil, &os.PathError{
			Op:   "open",
			Path: path,
			Err:  err,
		}
	}

	defer reader.Close()

	buffer, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, &os.PathError{
			Op:   "read",
			Path: path,
			Err:  err,
		}
	}

	return buffer, nil
}

func (zf *ZipFiler) ReadDir(path string) ([]filer.File, error) {
	zf.initTree()

	parts := strings.Split(path, "/")

	node := zf.tree
	for _, part := range parts {
		if part == "" {
			continue
		}
		node = node.children[part]
		if node == nil {
			return nil, &os.PathError{
				Op:   "readdir",
				Path: path,
				Err:  os.ErrNotExist,
			}
		}
	}

	if path != "" && !node.isDir() {
		return nil, &os.PathError{
			Op:   "readdir",
			Path: path,
			Err:  ErrNotDirectory,
		}
	}

	result := make([]filer.File, 0, len(node.children))
	for name, child := range node.children {
		result = append(result, filer.File{
			Name:  name,
			IsDir: child.isDir(),
		})
	}

	return result, nil
}

func (zf *ZipFiler) Close() {}

func (zf *ZipFiler) PathsAreAlwaysSlash() bool { return true }

func (zf *ZipFiler) initTree() {
	zf.onceInitTree.Do(func() {
		prefix := fmt.Sprintf("%s@%s", zf.Module.Name, zf.Module.Version.Original())
		root := &zipNode{children: map[string]*zipNode{}}
		for _, f := range zf.File {
			path := strings.Split(strings.TrimPrefix(f.Name, prefix), "/") // zip always has "/"
			node := root
			for _, part := range path {
				if part == "" {
					continue
				}
				child := node.children[part]
				if child == nil {
					child = &zipNode{children: map[string]*zipNode{}}
					node.children[part] = child
				}
				node = child
			}
			node.file = f
		}

		zf.tree = root
	})
}
