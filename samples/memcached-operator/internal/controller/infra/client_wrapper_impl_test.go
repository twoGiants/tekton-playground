package infra_test

import (
	"context"
	"errors"
	"testing"

	"example.com/m/v2/internal/controller/infra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func Test_ClientWrapper_ErrorResponses(t *testing.T) {
	testCases := []struct {
		expectedErr error
		operation   string
	}{
		{
			errors.New("Get error"),
			"Get",
		},
		{
			errors.New("StatusUpdate error"),
			"StatusUpdate",
		},
		{
			errors.New("Create error"),
			"Create",
		},
		{
			errors.New("Update error"),
			"Update",
		},
	}

	responses := make(map[string][]error)
	for _, tc := range testCases {
		responses[tc.operation] = []error{tc.expectedErr}
	}

	ctx := context.Background()

	k8ClientStub := infra.NewK8CliStub(responses, nil)

	if err := k8ClientStub.Get(ctx, types.NamespacedName{}, &corev1.Pod{}); err == nil {
		t.Errorf("expected %v, got nothing", responses["Get"])
	}

	if err := k8ClientStub.StatusUpdate(ctx, &corev1.Pod{}); err == nil {
		t.Errorf("expected %v, got nothing", responses["StatusUpdate"])
	}

	if err := k8ClientStub.Create(ctx, &corev1.Pod{}); err == nil {
		t.Errorf("expected %v, got nothing", responses["Create"])
	}

	if err := k8ClientStub.Update(ctx, &corev1.Pod{}); err == nil {
		t.Errorf("expected %v, got nothing", responses["Update"])
	}
}

func Test_ClientWrapper_NilResponses(t *testing.T) {
	ctx := context.Background()
	responses := make(map[string][]error)

	k8ClientStub := infra.NewK8CliStub(responses, nil)

	if err := k8ClientStub.Get(ctx, types.NamespacedName{}, &corev1.Pod{}); err != nil {
		t.Errorf("expected nil, got %v", err)
	}

	if err := k8ClientStub.StatusUpdate(ctx, &corev1.Pod{}); err != nil {
		t.Errorf("expected nil, got %v", err)
	}

	if err := k8ClientStub.Create(ctx, &corev1.Pod{}); err != nil {
		t.Errorf("expected nil, got %v", err)
	}

	if err := k8ClientStub.Update(ctx, &corev1.Pod{}); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}
