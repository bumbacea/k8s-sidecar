package handlers

import (
	v12 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

type cmHandler struct {
	globalHandler GenericHandler
}

func NewCmHandler(globalHandler GenericHandler) cache.ResourceEventHandler {
	return &cmHandler{globalHandler: globalHandler}
}

func (c *cmHandler) OnAdd(obj interface{}, isInInitialList bool) {
	cm := obj.(*v12.ConfigMap)
	c.globalHandler.OnAdd(cm.ObjectMeta, cm.Data, cm.BinaryData, isInInitialList)
}

func (c *cmHandler) OnUpdate(oldObj, newObj interface{}) {
	cmNew := newObj.(*v12.ConfigMap)
	cmOld := oldObj.(*v12.ConfigMap)
	c.globalHandler.OnUpdate(cmNew.ObjectMeta, cmOld.Data, cmOld.BinaryData, cmNew.Data, cmNew.BinaryData)
}

func (c *cmHandler) OnDelete(obj interface{}) {
	cm := obj.(*v12.ConfigMap)
	c.globalHandler.OnDelete(cm.ObjectMeta, cm.Data, cm.BinaryData)
}
