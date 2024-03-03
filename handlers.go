package main

import (
	"encoding/base64"
	"fmt"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"os"
	"path/filepath"
)

type secretsHandler struct {
	folder           string
	callback         func()
	defaultFileMode  uint32
	folderAnnotation string
}

func (s *secretsHandler) OnAdd(obj interface{}, isInInitialList bool) {
	secret := obj.(*v12.Secret)
	log.Printf("Adding secret %s/%s", secret.Namespace, secret.Name)
	if s.processConfigMap(secret.ObjectMeta, secret.StringData, secret.Data, false) {
		s.callback()
	}
}

func (s *secretsHandler) OnUpdate(oldObj, newObj interface{}) {
	secret := newObj.(*v12.Secret)
	log.Printf("Adding secret %s/%s", secret.Namespace, secret.Name)
	if s.processConfigMap(secret.ObjectMeta, secret.StringData, secret.Data, false) {
		s.callback()
	}
}

func (s *secretsHandler) OnDelete(obj interface{}) {
	secret := obj.(*v12.Secret)
	log.Printf("Adding secret %s/%s", secret.Namespace, secret.Name)
	if s.processConfigMap(secret.ObjectMeta, secret.StringData, secret.Data, true) {
		s.callback()
	}
}

// processConfigMap processes a ConfigMap
func (s *secretsHandler) processConfigMap(objectMeta v1.ObjectMeta, data map[string]string, dataBinary map[string][]byte, isDeleted bool) bool {
	log.Printf("Handling %s/%s", objectMeta.Namespace, objectMeta.Name)
	filesChanged := false

	folder := s.folder
	if val, ok := objectMeta.Annotations[s.folderAnnotation]; ok {
		folder = val
	}
	for key, value := range data {
		filesChanged = true
		filePath := filepath.Join(folder, key)
		err := os.WriteFile(filePath, []byte(value), os.FileMode(s.defaultFileMode))
		if err != nil {
			fmt.Println("Error writing file: ", err)
			continue
		}
	}

	for key, value := range dataBinary {
		filesChanged = true
		filePath := filepath.Join(folder, key)

		decoded, err := base64.StdEncoding.DecodeString(string(value))
		if err != nil {
			fmt.Println("Error decoding base64: ", err)
			continue
		}
		err = os.WriteFile(filePath, decoded, os.FileMode(s.defaultFileMode))
		if err != nil {
			fmt.Println("Error writing file: ", err)
			continue
		}
	}
	return filesChanged
}

type cmHandler struct {
	folder           string
	callback         func()
	defaultFileMode  uint32
	folderAnnotation string
}

func (c *cmHandler) OnAdd(obj interface{}, isInInitialList bool) {
	cm := obj.(*v12.ConfigMap)
	log.Printf("Adding cm %s/%s", cm.Namespace, cm.Name)
	if c.processConfigMap(cm.ObjectMeta, cm.Data, cm.BinaryData, false) {
		c.callback()
	}

}

func (c *cmHandler) OnUpdate(oldObj, newObj interface{}) {
	cm := newObj.(*v12.ConfigMap)
	if c.processConfigMap(cm.ObjectMeta, cm.Data, cm.BinaryData, false) {
		c.callback()
	}
}

func (c *cmHandler) OnDelete(obj interface{}) {
	cm := obj.(*v12.ConfigMap)
	log.Printf("Deleting cm %s/%s", cm.Namespace, cm.Name)
	if c.processConfigMap(cm.ObjectMeta, cm.Data, cm.BinaryData, true) {
		c.callback()
	}
}

// processConfigMap processes a ConfigMap
func (c *cmHandler) processConfigMap(objectMeta v1.ObjectMeta, data map[string]string, dataBinary map[string][]byte, isDeleted bool) bool {
	log.Printf("Handling %s/%s", objectMeta.Namespace, objectMeta.Name)
	filesChanged := false

	folder := c.folder
	if val, ok := objectMeta.Annotations[c.folderAnnotation]; ok {
		folder = val
	}
	for key, value := range data {
		filesChanged = true
		filePath := filepath.Join(folder, key)
		err := os.WriteFile(filePath, []byte(value), os.FileMode(c.defaultFileMode))
		if err != nil {
			fmt.Println("Error writing file: ", err)
			continue
		}
	}

	for key, value := range dataBinary {
		filesChanged = true
		filePath := filepath.Join(folder, key)

		decoded, err := base64.StdEncoding.DecodeString(string(value))
		if err != nil {
			fmt.Println("Error decoding base64: ", err)
			continue
		}
		err = os.WriteFile(filePath, decoded, os.FileMode(c.defaultFileMode))
		if err != nil {
			fmt.Println("Error writing file: ", err)
			continue
		}
	}
	return filesChanged
}
