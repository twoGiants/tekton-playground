package infra_test

import (
	"context"
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

// Test setup with test environment. Make sure to run 'make setup-envtest' to download
// the binaries needed for tests using the k8 cli.
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

	tnn, pod = tnnAndPod("non-existing-pod", "non-existing-ns")

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

func tnnAndPod(name, namespace string) (types.NamespacedName, *corev1.Pod) {
	tnn := types.NamespacedName{Name: name, Namespace: namespace}
	pod := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "main", Image: "busybox"}},
		},
	}
	return tnn, pod
}

// Test framework with Signature Shielding. If infra API changes the only place
// for updates in the tests will be in the helper functions.
//
// stubErrors: configurable responses for the stub
//
// cliType: stub|stubWithK8|impl; returns either real infrastructure or Embedded
// Stub
func newK8Cli(stubErrors infra.StubErrors, cliType string) *infra.K8CliImpl {
	switch cliType {
	case "stub":
		return infra.NewK8CliStub(stubErrors, nil)
	case "stubWithK8":
		return infra.NewK8CliStub(stubErrors, k8TestCli)
	case "impl":
		return infra.NewK8CliImpl(k8TestCli)
	default:
		return infra.NewK8CliStub(stubErrors, nil)
	}
}

type options struct {
	cmd        string
	cliType    string
	stubErrors infra.StubErrors
	pod        *corev1.Pod
	tnn        *types.NamespacedName
}

func runK8Cli(ctx context.Context, opt options) error {
	if opt.pod == nil {
		panic(fmt.Errorf("pod must be provided"))
	}

	k8 := newK8Cli(opt.stubErrors, opt.cliType)

	switch opt.cmd {
	case "Get":
		if opt.tnn == nil {
			panic(fmt.Errorf("provide a NamespacedName when using 'Get'"))
		}
		return k8.Get(ctx, *opt.tnn, opt.pod)
	case "StatusUpdate":
		return k8.StatusUpdate(ctx, opt.pod)
	case "Create":
		return k8.Create(ctx, opt.pod)
	case "Update":
		return k8.Update(ctx, opt.pod)
	default:
		panic(fmt.Errorf("unknown command: %s", opt.cmd))
	}
}

// deprecated
func k8Get(stubErrors infra.StubErrors, cliType string) error {
	k8 := newK8Cli(stubErrors, cliType)
	return k8.Get(ctx, tnn, pod)
}

// deprecated
func k8StatusUpdate(stubErrors infra.StubErrors, cliType string) error {
	k8 := newK8Cli(stubErrors, cliType)
	return k8.StatusUpdate(ctx, pod)
}

// deprecated
func k8Create(stubErrors infra.StubErrors, cliType string) error {
	k8 := newK8Cli(stubErrors, cliType)
	return k8.Create(ctx, pod)
}

// deprecated
func k8Update(stubErrors infra.StubErrors, cliType string) error {
	k8 := newK8Cli(stubErrors, cliType)
	return k8.Update(ctx, pod)
}

func createStubErrors(n int) infra.StubErrors {
	if n == 0 {
		return infra.StubErrors{
			"Get":          {nil},
			"StatusUpdate": {nil},
			"Create":       {nil},
			"Update":       {nil},
		}
	}

	result := make(infra.StubErrors)
	for i := 1; i <= n; i++ {
		result["Get"] = append(result["Get"], fmt.Errorf("Get error %d", i))
		result["StatusUpdate"] = append(result["StatusUpdate"], fmt.Errorf("StatusUpdate error %d", i))
		result["Create"] = append(result["Create"], fmt.Errorf("Create error %d", i))
		result["Update"] = append(result["Update"], fmt.Errorf("Update error %d", i))
	}

	return result
}

func assertNotFound(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Error("expected err, got nothing")
	}
	if !apierrors.IsNotFound(err) {
		t.Errorf("expected NotFoundError, got %v", err)
	}
}
