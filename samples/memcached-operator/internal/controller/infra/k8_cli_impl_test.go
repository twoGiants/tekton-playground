package infra_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	cachev1alpha1 "example.com/m/v2/api/v1alpha1"
	"example.com/m/v2/internal/controller/infra"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	ctx       context.Context
	k8TestCli client.Client
	tnn       types.NamespacedName
	pod       *corev1.Pod
)

func TestMain(m *testing.M) {
	ctx = context.Background()

	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	if binaryAssetsDir() == "" {
		fmt.Printf("failed to load test env k8s binary, make sure binaries are downloaded with 'make setup-envtest'")
		os.Exit(1)
	}
	testEnv.BinaryAssetsDirectory = binaryAssetsDir()

	if err := cachev1alpha1.AddToScheme(scheme.Scheme); err != nil {
		fmt.Printf("failed to add scheme: %v\n", err)
		os.Exit(1)
	}

	cfg, err := testEnv.Start()
	if err != nil {
		fmt.Printf("failed to start test environment: %v\n", err)
		os.Exit(1)
	}

	if k8TestCli, err = client.New(cfg, client.Options{Scheme: scheme.Scheme}); err != nil {
		fmt.Printf("failed to create k8s client: %v\n", err)
		os.Exit(1)
	}

	if k8TestCli == nil {
		fmt.Println("k8sClient is nil")
		os.Exit(1)
	}

	tnn, pod = tnnAndPod()

	code := m.Run()

	if err = testEnv.Stop(); err != nil {
		fmt.Printf("failed to stop test environment: %v\n", err)
		os.Exit(1)
	}

	os.Exit(code)
}

func binaryAssetsDir() string {
	basePath := filepath.Join("..", "..", "..", "bin", "k8s")
	entries, err := os.ReadDir(basePath)
	if err != nil {
		fmt.Printf("failed to read directory %s", basePath)
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(basePath, entry.Name())
		}
	}
	return ""
}

func tnnAndPod() (types.NamespacedName, *corev1.Pod) {
	name := "not-existing-pod"
	namespace := "not-existing-ns"
	tnn := types.NamespacedName{Name: name, Namespace: namespace}
	pod := &corev1.Pod{ObjectMeta: v1.ObjectMeta{Name: name, Namespace: namespace}}
	return tnn, pod
}

func Test_K8CliStub_errorResponses(t *testing.T) {
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

	k8 := infra.NewK8CliStub(responses, nil)

	if err := k8.Get(ctx, tnn, pod); err == nil {
		t.Errorf("expected %v, got nothing", responses["Get"])
	}

	if err := k8.StatusUpdate(ctx, pod); err == nil {
		t.Errorf("expected %v, got nothing", responses["StatusUpdate"])
	}

	if err := k8.Create(ctx, pod); err == nil {
		t.Errorf("expected %v, got nothing", responses["Create"])
	}

	if err := k8.Update(ctx, pod); err == nil {
		t.Errorf("expected %v, got nothing", responses["Update"])
	}
}

func Test_K8CliStub_nilResponses(t *testing.T) {
	responses := make(map[string][]error)

	k8 := infra.NewK8CliStub(responses, nil)

	if err := k8.Get(ctx, tnn, pod); err != nil {
		t.Errorf("expected nil, got %v", err)
	}

	if err := k8.StatusUpdate(ctx, pod); err != nil {
		t.Errorf("expected nil, got %v", err)
	}

	if err := k8.Create(ctx, pod); err != nil {
		t.Errorf("expected nil, got %v", err)
	}

	if err := k8.Update(ctx, pod); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func Test_K8CliStub_nilAfterError(t *testing.T) {
	errMap := map[string][]error{"Get": {errors.New("Get error")}}

	k8 := infra.NewK8CliStub(errMap, nil)

	if err := k8.Get(ctx, tnn, pod); err == nil {
		t.Errorf("expected %v, got nothing", errMap["Get"])
	}

	if err := k8.Get(ctx, tnn, pod); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func Test_K8CliStub_multiErrors(t *testing.T) {
	errMap := map[string][]error{"Get": {errors.New("err 1"), errors.New("err 2")}}

	k8 := infra.NewK8CliStub(errMap, nil)

	if err := k8.Get(ctx, tnn, pod); err == nil {
		t.Errorf("expected %v, got nothing", errMap["Get"][0])
	}

	if err := k8.Get(ctx, tnn, pod); err == nil {
		t.Errorf("expected %v, got nothing", errMap["Get"][1])
	}

	if err := k8.Get(ctx, tnn, pod); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func Test_K8CliStub_realK8Cli_withError(t *testing.T) {
	errMap := make(map[string][]error)

	k8 := infra.NewK8CliStub(errMap, k8TestCli)

	err := k8.Get(ctx, tnn, pod)
	assertNotFound(t, err)

	err = k8.StatusUpdate(ctx, pod)
	assertNotFound(t, err)

	err = k8.Create(ctx, pod)
	assertNotFound(t, err)

	err = k8.Update(ctx, pod)
	assertNotFound(t, err)
}

func assertNotFound(t *testing.T, err error) {
	if err == nil {
		t.Error("expected err, got nothing")
	}
	if !apierrors.IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %v", err)
	}
}

func Test_K8Cli_forwardsError(t *testing.T) {
	k8 := infra.NewK8CliImpl(k8TestCli)

	err := k8.Get(ctx, tnn, pod)
	assertNotFound(t, err)

	err = k8.StatusUpdate(ctx, pod)
	assertNotFound(t, err)

	err = k8.Create(ctx, pod)
	assertNotFound(t, err)

	err = k8.Update(ctx, pod)
	assertNotFound(t, err)
}
