package onepassword

import (
	"context"
	"fmt"
	"testing"

	"github.com/1Password/onepassword-operator/pkg/mocks"

	"github.com/1Password/connect-sdk-go/onepassword"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	name        = "test-workload"
	namespace   = "default"
	vaultId     = "hfnjvi6aymbsnfc2xeeoheizda"
	itemId      = "nwrhuano7bcwddcviubpp4mhfq"
	username    = "test-user"
	password    = "QmHumKc$mUeEem7caHtbaBaJ"
	userKey     = "username"
	passKey     = "password"
	itemVersion = 123
)

type testUpdateSecretTask struct {
	testName                 string
	existingWorkload         runtime.Object
	existingNamespace        *corev1.Namespace
	existingSecret           *corev1.Secret
	expectedError            error
	expectedResultSecret     *corev1.Secret
	expectedEvents           []string
	opItem                   map[string]string
	expectedRestart          bool
	globalAutoRestartEnabled bool
}

var (
	expectedSecretData = map[string][]byte{
		"password": []byte(password),
		"username": []byte(username),
	}
	itemPath = fmt.Sprintf("vaults/%v/items/%v", vaultId, itemId)
)

var defaultNamespace = &corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: namespace,
	},
}

var tests = []testUpdateSecretTask{
	{
		testName:          "Test unrelated deployment is not restarted with an updated secret",
		existingNamespace: defaultNamespace,
		existingWorkload: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					NameAnnotation:     "unlrelated secret",
					ItemPathAnnotation: itemPath,
				},
			},
		},
		existingSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:  "old version",
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		expectedError: nil,
		expectedResultSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:  fmt.Sprint(itemVersion),
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedRestart:          false,
		globalAutoRestartEnabled: true,
	},
	{
		testName:          "OP item has new version. Secret needs update. Deployment is restarted based on containers",
		existingNamespace: defaultNamespace,
		existingWorkload: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"external-annotation": "some-value"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Env: []corev1.EnvVar{
									{
										Name: name,
										ValueFrom: &corev1.EnvVarSource{
											SecretKeyRef: &corev1.SecretKeySelector{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: name,
												},
												Key: passKey,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		existingSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:  "old version",
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		expectedError: nil,
		expectedResultSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:  fmt.Sprint(itemVersion),
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedRestart:          true,
		globalAutoRestartEnabled: true,
	},
	{
		testName:          "OP item has new version. Secret needs update. Deployment is restarted based on annotation",
		existingNamespace: defaultNamespace,
		existingWorkload: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					ItemPathAnnotation: itemPath,
					NameAnnotation:     name,
				},
			},
		},
		existingSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:  "old version",
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		expectedError: nil,
		expectedResultSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:  fmt.Sprint(itemVersion),
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedRestart:          true,
		globalAutoRestartEnabled: true,
	},
	{
		testName:          "OP item has new version. Secret needs update. Deployment is restarted based on volume",
		existingNamespace: defaultNamespace,
		existingWorkload: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"external-annotation": "some-value"},
					},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: name,
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										SecretName: name,
									},
								},
							},
						},
					},
				},
			},
		},
		existingSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:  "old version",
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		expectedError: nil,
		expectedResultSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:  fmt.Sprint(itemVersion),
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedRestart:          true,
		globalAutoRestartEnabled: true,
	},
	{
		testName:          "No secrets need update. No deployment is restarted",
		existingNamespace: defaultNamespace,
		existingWorkload: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					ItemPathAnnotation: itemPath,
					NameAnnotation:     name,
				},
			},
		},
		existingSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:  fmt.Sprint(itemVersion),
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		expectedError: nil,
		expectedResultSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:  fmt.Sprint(itemVersion),
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedRestart:          false,
		globalAutoRestartEnabled: true,
	},
	{
		testName: `Deployment is not restarted when no auto restart is set to true for all
		deployments and is not overwritten by by a namespace or deployment annotation`,
		existingNamespace: defaultNamespace,
		existingWorkload: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"external-annotation": "some-value"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Env: []corev1.EnvVar{
									{
										Name: name,
										ValueFrom: &corev1.EnvVarSource{
											SecretKeyRef: &corev1.SecretKeySelector{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: name,
												},
												Key: passKey,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		existingSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:  "old version",
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		expectedError: nil,
		expectedResultSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:  fmt.Sprint(itemVersion),
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedRestart:          false,
		globalAutoRestartEnabled: false,
	},
	{
		testName:          `Secret autostart true value takes precedence over false deployment value`,
		existingNamespace: defaultNamespace,
		existingWorkload: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					AutoRestartWorkloadAnnotation: "false",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"external-annotation": "some-value"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Env: []corev1.EnvVar{
									{
										Name: name,
										ValueFrom: &corev1.EnvVarSource{
											SecretKeyRef: &corev1.SecretKeySelector{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: name,
												},
												Key: passKey,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		existingSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:             "old version",
					ItemPathAnnotation:            itemPath,
					AutoRestartWorkloadAnnotation: "true",
				},
			},
			Data: expectedSecretData,
		},
		expectedError: nil,
		expectedResultSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:             fmt.Sprint(itemVersion),
					ItemPathAnnotation:            itemPath,
					AutoRestartWorkloadAnnotation: "true",
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedRestart:          true,
		globalAutoRestartEnabled: false,
	},
	{
		testName:          `Secret autostart true value takes precedence over false deployment value`,
		existingNamespace: defaultNamespace,
		existingWorkload: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					AutoRestartWorkloadAnnotation: "true",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"external-annotation": "some-value"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Env: []corev1.EnvVar{
									{
										Name: name,
										ValueFrom: &corev1.EnvVarSource{
											SecretKeyRef: &corev1.SecretKeySelector{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: name,
												},
												Key: passKey,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		existingSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:             "old version",
					ItemPathAnnotation:            itemPath,
					AutoRestartWorkloadAnnotation: "false",
				},
			},
			Data: expectedSecretData,
		},
		expectedError: nil,
		expectedResultSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:             fmt.Sprint(itemVersion),
					ItemPathAnnotation:            itemPath,
					AutoRestartWorkloadAnnotation: "false",
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedRestart:          false,
		globalAutoRestartEnabled: true,
	},
	{
		testName:          `Deployment autostart true value takes precedence over false global auto restart value`,
		existingNamespace: defaultNamespace,
		existingWorkload: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					AutoRestartWorkloadAnnotation: "true",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"external-annotation": "some-value"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Env: []corev1.EnvVar{
									{
										Name: name,
										ValueFrom: &corev1.EnvVarSource{
											SecretKeyRef: &corev1.SecretKeySelector{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: name,
												},
												Key: passKey,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		existingSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:  "old version",
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		expectedError: nil,
		expectedResultSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:  fmt.Sprint(itemVersion),
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedRestart:          true,
		globalAutoRestartEnabled: false,
	},
	{
		testName: `Deployment autostart false value takes precedence over false global auto restart value,
		 and true namespace value.`,
		existingNamespace: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
				Annotations: map[string]string{
					AutoRestartWorkloadAnnotation: "true",
				},
			},
		},
		existingWorkload: &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					AutoRestartWorkloadAnnotation: "false",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"external-annotation": "some-value"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Env: []corev1.EnvVar{
									{
										Name: name,
										ValueFrom: &corev1.EnvVarSource{
											SecretKeyRef: &corev1.SecretKeySelector{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: name,
												},
												Key: passKey,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		existingSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:  "old version",
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		expectedError: nil,
		expectedResultSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:  fmt.Sprint(itemVersion),
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedRestart:          false,
		globalAutoRestartEnabled: false,
	},
	{
		testName: `Namespace autostart true value takes precedence over false global auto restart value`,
		existingNamespace: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
				Annotations: map[string]string{
					AutoRestartWorkloadAnnotation: "true",
				},
			},
		},
		existingWorkload: &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: appsv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"external-annotation": "some-value"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Env: []corev1.EnvVar{
									{
										Name: name,
										ValueFrom: &corev1.EnvVarSource{
											SecretKeyRef: &corev1.SecretKeySelector{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: name,
												},
												Key: passKey,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		existingSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:  "old version",
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		expectedError: nil,
		expectedResultSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:  fmt.Sprint(itemVersion),
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedRestart:          true,
		globalAutoRestartEnabled: false,
	},
}

func TestUpdateSecretHandler(t *testing.T) {
	for _, testData := range tests {
		t.Run(testData.testName, func(t *testing.T) {
			// Register workload type with the scheme.
			s := scheme.Scheme
			s.AddKnownTypes(appsv1.SchemeGroupVersion, testData.existingWorkload)

			// Build fake client with relevant runtime objects.
			objs := []runtime.Object{testData.existingWorkload, testData.existingNamespace}
			if testData.existingSecret != nil {
				objs = append(objs, testData.existingSecret)
			}

			fakeClient := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()

			opConnectClient := &mocks.TestClient{}
			mocks.DoGetItemFunc = func(uuid string, vaultUUID string) (*onepassword.Item, error) {
				return &onepassword.Item{
					ID:      uuid,
					Vault:   onepassword.ItemVault{ID: vaultUUID},
					Fields:  generateFields(testData.opItem["username"], testData.opItem["password"]),
					Version: itemVersion,
				}, nil
			}

			h := &SecretUpdateHandler{
				client:                       fakeClient,
				opConnectClient:              opConnectClient,
				autoRestartWorkloadsGlobally: testData.globalAutoRestartEnabled,
			}

			err := h.UpdateKubernetesSecretsTask()
			assert.Equal(t, testData.expectedError, err)

			// Check the resulting secret
			var expectedSecretName string
			if testData.expectedResultSecret == nil {
				expectedSecretName = testData.existingWorkload.(client.Object).GetName()
			} else {
				expectedSecretName = testData.expectedResultSecret.Name
			}

			secret := &corev1.Secret{}
			err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: expectedSecretName, Namespace: namespace}, secret)

			if testData.expectedResultSecret == nil {
				assert.Error(t, err)
				assert.True(t, errors2.IsNotFound(err))
			} else {
				assert.Equal(t, testData.expectedResultSecret.Data, secret.Data)
				assert.Equal(t, testData.expectedResultSecret.Name, secret.Name)
				assert.Equal(t, testData.expectedResultSecret.Type, secret.Type)
				assert.Equal(t, testData.expectedResultSecret.Annotations[VersionAnnotation], secret.Annotations[VersionAnnotation])
			}

			// Check for restart annotation
			objKey := client.ObjectKey{
				Name:      testData.existingWorkload.(client.Object).GetName(),
				Namespace: testData.existingWorkload.(client.Object).GetNamespace(),
			}

			// Fetch updated workload
			fetchedWorkload := testData.existingWorkload.DeepCopyObject().(client.Object)
			err = fakeClient.Get(context.TODO(), objKey, fetchedWorkload)
			assert.NoError(t, err)

			newAnnotations := getPodTemplateAnnotations(fetchedWorkload)
			oldAnnotations := getPodTemplateAnnotations(testData.existingWorkload)

			hasRestartAnnotation := newAnnotations[RestartAnnotation] != ""

			if testData.expectedRestart {
				assert.True(t, hasRestartAnnotation, "Expected workload to restart but it did not")
			} else {
				assert.False(t, hasRestartAnnotation, "Workload was restarted but should not have been")
			}

			for k, v := range oldAnnotations {
				if k == RestartAnnotation {
					continue
				}
				actual, ok := newAnnotations[k]
				if assert.True(t, ok, "Expected annotation %q to be preserved", k) {
					assert.Equal(t, v, actual, "Annotation value changed for %q", k)
				}
			}
		})
	}
}
func getPodTemplateAnnotations(obj runtime.Object) map[string]string {
	switch o := obj.(type) {
	case *appsv1.Deployment:
		return o.Spec.Template.Annotations
	case *appsv1.DaemonSet:
		return o.Spec.Template.Annotations
	default:
		return map[string]string{}
	}
}

func TestIsUpdatedSecret(t *testing.T) {
	secretName := "test-secret"
	updatedSecrets := map[string]*corev1.Secret{
		"some_secret": {},
	}
	assert.False(t, isUpdatedSecret(secretName, updatedSecrets))

	updatedSecrets[secretName] = &corev1.Secret{}
	assert.True(t, isUpdatedSecret(secretName, updatedSecrets))
}

func generateFields(username, password string) []*onepassword.ItemField {
	fields := []*onepassword.ItemField{
		{
			Label: "username",
			Value: username,
		},
		{
			Label: "password",
			Value: password,
		},
	}
	return fields
}
