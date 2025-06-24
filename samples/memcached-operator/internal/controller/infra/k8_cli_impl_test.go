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

func k8Get(stubErrors infra.StubErrors, cliType string) error {
	k8 := newK8Cli(stubErrors, cliType)
	return k8.Get(ctx, tnn, pod)
}

func k8StatusUpdate(stubErrors infra.StubErrors, cliType string) error {
	k8 := newK8Cli(stubErrors, cliType)
	return k8.StatusUpdate(ctx, pod)
}

func k8Create(stubErrors infra.StubErrors, cliType string) error {
	k8 := newK8Cli(stubErrors, cliType)
	return k8.Create(ctx, pod)
}

func k8Update(stubErrors infra.StubErrors, cliType string) error {
	k8 := newK8Cli(stubErrors, cliType)
	return k8.Update(ctx, pod)
}

// Start unit tests
func Test_K8Cli_stubWithConfigurableResponses(t *testing.T) {
	testCases := []struct {
		name           string
		stubErrors     infra.StubErrors
		expectedErrors infra.StubErrors
	}{{
		name: "one error for each method",
		stubErrors: infra.StubErrors{
			"Get":          {errors.New("Get error")},
			"StatusUpdate": {errors.New("StatusUpdate error")},
			"Create":       {errors.New("Create error")},
			"Update":       {errors.New("Update error")},
		},
		expectedErrors: infra.StubErrors{
			"Get":          {errors.New("Get error")},
			"StatusUpdate": {errors.New("StatusUpdate error")},
			"Create":       {errors.New("Create error")},
			"Update":       {errors.New("Update error")},
		},
	}}

	var err error
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err = k8Get(tc.stubErrors, "stub"); err == nil {
				t.Errorf("expected %s, got nothing", tc.expectedErrors["Get"][0].Error())
			}
			if err.Error() != tc.expectedErrors["Get"][0].Error() {
				t.Errorf("expected %s, got %s", tc.expectedErrors["Get"][0].Error(), err.Error())
			}

			if err = k8StatusUpdate(tc.stubErrors, "stub"); err == nil {
				t.Errorf("expected %s, got nothing", tc.expectedErrors["StatusUpdate"][0].Error())
			}
			if err.Error() != tc.expectedErrors["StatusUpdate"][0].Error() {
				t.Errorf("expected %s, got %s", tc.expectedErrors["StatusUpdate"][0].Error(), err.Error())
			}

			if err = k8Create(tc.stubErrors, "stub"); err == nil {
				t.Errorf("expected %s, got nothing", tc.expectedErrors["Create"][0].Error())
			}
			if err.Error() != tc.expectedErrors["Create"][0].Error() {
				t.Errorf("expected %s, got %s", tc.expectedErrors["Create"][0].Error(), err.Error())
			}

			if err = k8Update(tc.stubErrors, "stub"); err == nil {
				t.Errorf("expected %s, got nothing", tc.expectedErrors["Update"][0].Error())
			}
			if err.Error() != tc.expectedErrors["Update"][0].Error() {
				t.Errorf("expected %s, got %s", tc.expectedErrors["Update"][0].Error(), err.Error())
			}
		})
	}
}

func Test_K8Cli_stubWithMultiErrors(t *testing.T) {
	stubErrors := infra.StubErrors{"Get": {errors.New("err 1"), errors.New("err 2")}}

	if err := k8Get(stubErrors, "stub"); err == nil {
		t.Errorf("expected %v, got nothing", stubErrors["Get"][0])
	}

	if err := k8Get(stubErrors, "stub"); err == nil {
		t.Errorf("expected %v, got nothing", stubErrors["Get"][1])
	}

	if err := k8Get(stubErrors, "stub"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func Test_K8Cli_stubWithNilResponses(t *testing.T) {
	testCases := []struct {
		name          string
		stubErrors    infra.StubErrors
		expectedError error
	}{{
		name:          "no errors provided for stub",
		expectedError: nil,
	}, {
		name:          "error for non existing method name",
		stubErrors:    infra.StubErrors{"NonExisting": {errors.New("NonExisting error")}},
		expectedError: nil,
	}, {
		name: "nil error",
		stubErrors: infra.StubErrors{
			"Get":          {nil},
			"StatusUpdate": {nil},
			"Create":       {nil},
			"Update":       {nil},
		},
		expectedError: nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := k8Get(tc.stubErrors, "stub"); err != tc.expectedError {
				t.Errorf("expected %v, got %v", tc.expectedError, err)
			}

			if err := k8StatusUpdate(tc.stubErrors, "stub"); err != tc.expectedError {
				t.Errorf("expected %v, got %v", tc.expectedError, err)
			}

			if err := k8Create(tc.stubErrors, "stub"); err != tc.expectedError {
				t.Errorf("expected %v, got %v", tc.expectedError, err)
			}

			if err := k8Update(tc.stubErrors, "stub"); err != tc.expectedError {
				t.Errorf("expected %v, got %v", tc.expectedError, err)
			}
		})
	}
}

func Test_K8Cli_stubWithNilAfterError(t *testing.T) {
	stubErrors := infra.StubErrors{
		"Get":          {errors.New("Get error")},
		"StatusUpdate": {errors.New("StatusUpdate error")},
		"Create":       {errors.New("Create error")},
		"Update":       {errors.New("Update error")},
	}

	if err := k8Get(stubErrors, "stub"); err == nil {
		t.Errorf("expected %v, got nothing", stubErrors["Get"])
	}
	if err := k8Get(stubErrors, "stub"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}

	if err := k8StatusUpdate(stubErrors, "stub"); err == nil {
		t.Errorf("expected %v, got nothing", stubErrors["StatusUpdate"])
	}
	if err := k8StatusUpdate(stubErrors, "stub"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}

	if err := k8Create(stubErrors, "stub"); err == nil {
		t.Errorf("expected %v, got nothing", stubErrors["Create"])
	}
	if err := k8Create(stubErrors, "stub"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}

	if err := k8Update(stubErrors, "stub"); err == nil {
		t.Errorf("expected %v, got nothing", stubErrors["Update"])
	}
	if err := k8Update(stubErrors, "stub"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func Test_K8Cli_stubWithRealK8CliWithError(t *testing.T) {
	err := k8Get(nil, "stubWithK8")
	assertNotFound(t, err)

	err = k8StatusUpdate(nil, "stubWithK8")
	assertNotFound(t, err)

	err = k8Create(nil, "stubWithK8")
	assertNotFound(t, err)

	err = k8Update(nil, "stubWithK8")
	assertNotFound(t, err)
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

func Test_K8Cli_errorPropagation(t *testing.T) {
	testCases := []struct {
		name    string
		cliType string
	}{{
		name:    "actual implementation propagates k8 cli error",
		cliType: "impl",
	}, {
		name:    "stub with real k8 cli propagates error",
		cliType: "stubWithK8",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := k8Get(nil, tc.cliType)
			assertNotFound(t, err)

			err = k8StatusUpdate(nil, tc.cliType)
			assertNotFound(t, err)

			err = k8Create(nil, tc.cliType)
			assertNotFound(t, err)

			err = k8Update(nil, tc.cliType)
			assertNotFound(t, err)
		})
	}
}
