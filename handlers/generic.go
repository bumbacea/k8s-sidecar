package handlers

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"os"
	"path/filepath"
	"sync"
)

func (g *genericHandlerImpl) writeData(path string, data []byte) (bool, error) {
	g.sm.Lock()
	defer g.sm.Unlock()
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return true, os.WriteFile(path, data, os.FileMode(g.defaultFileMode))
	}
	current, err := os.ReadFile(path)
	if err != nil {
		log.Printf("failed to read previous version of the file")
	}
	if bytes.Equal(current, data) {
		return false, nil
	}
	return true, os.WriteFile(path, data, os.FileMode(g.defaultFileMode))
}

type genericHandlerImpl struct {
	folder           string
	callback         func()
	defaultFileMode  uint32
	folderAnnotation string
	uniqFilenames    bool
	sm               *sync.Mutex
}

func NewGenericHandlerImpl(folder string, callback func(), defaultFileMode uint32, folderAnnotation string, uniqFilenames bool) GenericHandler {
	return &genericHandlerImpl{
		folder:           folder,
		callback:         callback,
		defaultFileMode:  defaultFileMode,
		folderAnnotation: folderAnnotation,
		uniqFilenames:    uniqFilenames,
		sm:               &sync.Mutex{},
	}
}

func (g *genericHandlerImpl) OnAdd(meta v1.ObjectMeta, data map[string]string, binaryData map[string][]byte, isInInitialList bool) {
	if g.processData(meta, data, binaryData, isInInitialList) {
		g.callback()
	}
}

func (g *genericHandlerImpl) OnUpdate(meta v1.ObjectMeta, _ map[string]string, _ map[string][]byte, data map[string]string, binaryData map[string][]byte) {
	if g.processData(meta, data, binaryData, false) {
		g.callback()
	}
}

func (g *genericHandlerImpl) OnDelete(meta v1.ObjectMeta, data map[string]string, dataBinary map[string][]byte) {
	deletedFiles := false
	for s := range data {
		path := g.computePath(meta, s)
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			e := os.Remove(path)
			if e != nil {
				log.Printf("failed to delete file %s: %s", err)
				continue
			}
			deletedFiles = true
		}
	}
	for s := range dataBinary {
		path := g.computePath(meta, s)
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			e := os.Remove(path)
			if e != nil {
				log.Printf("failed to delete file %s: %s", err)
				continue
			}
			deletedFiles = true
		}
	}
	if deletedFiles {
		g.callback()
	}
}

func (g *genericHandlerImpl) processData(objectMeta v1.ObjectMeta, data map[string]string, dataBinary map[string][]byte, isInInitialList bool) bool {
	log.Printf("Handling cm %s/%s", objectMeta.Namespace, objectMeta.Name)
	filesChanged := false
	for key, value := range data {
		changed, err := g.writeData(g.computePath(objectMeta, key), []byte(value))
		if err != nil {
			fmt.Println("Error writing file: ", err)
			continue
		}
		if changed == true {
			filesChanged = true
		}
	}

	for key, value := range dataBinary {
		decoded, err := base64.StdEncoding.DecodeString(string(value))
		if err != nil {
			fmt.Println("Error decoding base64: ", err)
			continue
		}
		changed, err := g.writeData(g.computePath(objectMeta, key), decoded)
		if err != nil {
			fmt.Println("Error writing file: ", err)
			continue
		}
		if changed == true {
			filesChanged = true
		}
	}

	return filesChanged
}

func (g *genericHandlerImpl) computePath(objectMeta v1.ObjectMeta, key string) string {
	folder := g.folder
	if val, ok := objectMeta.Annotations[g.folderAnnotation]; ok {
		folder = val
	}
	filename := key
	if g.uniqFilenames {
		filename = fmt.Sprintf("namespace_%s.resource_%s.%s", objectMeta.Name, objectMeta.Name, key)
	}
	return filepath.Join(folder, filename)
}

type GenericHandler interface {
	OnAdd(meta v1.ObjectMeta, data map[string]string, binaryData map[string][]byte, isInInitialList bool)
	OnUpdate(meta v1.ObjectMeta, oldData map[string]string, oldBinaryData map[string][]byte, data map[string]string, binaryData map[string][]byte)
	OnDelete(meta v1.ObjectMeta, data map[string]string, binaryData map[string][]byte)
}
