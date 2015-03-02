package api

import (
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
)

func init() {
	api.Scheme.AddKnownTypes("",
		&Build{},
		&BuildList{},
		&BuildConfig{},
		&BuildConfigList{},
	)
}

func (*Build) IsAnAPIObject()           {}
func (*BuildList) IsAnAPIObject()       {}
func (*BuildConfig) IsAnAPIObject()     {}
func (*BuildConfigList) IsAnAPIObject() {}
