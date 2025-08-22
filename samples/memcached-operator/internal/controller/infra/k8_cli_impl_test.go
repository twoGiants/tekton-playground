package infra_test

import (
	"errors"
	"testing"

	"example.com/m/v2/internal/controller/infra"
	v1 "k8s.io/api/core/v1"
)

func Test_K8Cli_stubWithConfigurableResponses(t *testing.T) {
	testCases := []struct {
		name           string
		stubErrors     infra.StubErrors
		expectedErrors infra.StubErrors
	}{{
		name:           "one error for each method",
		stubErrors:     createStubErrors(1),
		expectedErrors: createStubErrors(1),
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
		name:          "nil error",
		stubErrors:    createStubErrors(0),
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
	stubErrors := createStubErrors(1)

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

func Test_K8Cli_commandPropagation(t *testing.T) {
	testCases := []struct {
		name,
		podName,
		imageUpdate,
		cliType string
		statusPhase v1.PodPhase
	}{{
		name:        "actual implementation propagates k8 cli command",
		podName:     "existing-pod",
		imageUpdate: "ubuntu",
		statusPhase: "Running",
		cliType:     "impl",
	}, {
		name:        "stub with real k8 cli propagates cli",
		podName:     "second-existing-pod",
		imageUpdate: "fedora",
		statusPhase: "Pending",
		cliType:     "stubWithK8",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tnn, pod = tnnAndPod(tc.podName, "default")

			// create command propagation
			if err := k8Create(nil, tc.cliType); err != nil {
				t.Errorf("unexpected error creating pod %v", err)
			}

			// update command propagation
			pod.Spec.Containers[0].Image = tc.imageUpdate
			if err := k8Update(nil, tc.cliType); err != nil {
				t.Errorf("unexpected error updating pod %v", err)
			}

			// status update command propagation
			pod.Status.Phase = tc.statusPhase
			if err := k8StatusUpdate(nil, tc.cliType); err != nil {
				t.Errorf("unexpected error updating pod status %v", err)
			}

			// get command propagation
			_, pod = tnnAndPod("existing-pod", "default")
			if err := k8Get(nil, tc.cliType); err != nil {
				t.Errorf("unexpected error getting pod %v", err)
			}
			if pod.Spec.Containers[0].Image != tc.imageUpdate {
				t.Errorf("expected container image %s, got %s", tc.imageUpdate, pod.Spec.Containers[0].Image)
			}
			if pod.Status.Phase != tc.statusPhase {
				t.Errorf("expected pod status %s, got %s", tc.statusPhase, pod.Status.Phase)
			}
		})
	}
}
