package handlers

import (
	v12 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

type secretsHandler struct {
	globalHandler GenericHandler
}

func NewSecretsHandler(globalHandler GenericHandler) cache.ResourceEventHandler {
	return &secretsHandler{globalHandler: globalHandler}
}

func (s *secretsHandler) OnAdd(obj interface{}, isInInitialList bool) {
	secret := obj.(*v12.Secret)
	s.globalHandler.OnAdd(secret.ObjectMeta, secret.StringData, secret.Data, isInInitialList)
}

func (s *secretsHandler) OnUpdate(oldObj, newObj interface{}) {
	newSecret := newObj.(*v12.Secret)
	oldSecret := oldObj.(*v12.Secret)
	s.globalHandler.OnUpdate(newSecret.ObjectMeta, oldSecret.StringData, oldSecret.Data, newSecret.StringData, newSecret.Data)
}

func (s *secretsHandler) OnDelete(obj interface{}) {
	secret := obj.(*v12.Secret)
	s.globalHandler.OnDelete(secret.ObjectMeta, secret.StringData, secret.Data)
}
