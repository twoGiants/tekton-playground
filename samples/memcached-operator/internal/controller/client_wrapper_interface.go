package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type K8CliWrapper interface {
	Get(context.Context, types.NamespacedName, client.Object) error
	StatusUpdate(context.Context, client.Object) error
	Create(context.Context, client.Object) error
	Update(context.Context, client.Object) error
}
