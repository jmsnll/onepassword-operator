package controller

const (
	daemonSetKind       = "DaemonSet"
	daemonSetAPIVersion = "v1"
	daemonSetName       = "test-daemonset"
)

//var _ = Describe("DaemonSet controller", func() {
//	var ctx context.Context
//	var daemonSetKey types.NamespacedName
//	var secretKey types.NamespacedName
//	var daemonSetResource *appsv1.DaemonSet
//	createdSecret := &v1.Secret{}
//
//	makeDaemonSet := func() {
//		ctx = context.Background()
//
//		daemonSetKey = types.NamespacedName{
//			Name:      daemonSetName,
//			Namespace: namespace,
//		}
//
//		secretKey = types.NamespacedName{
//			Name:      item1.Name,
//			Namespace: namespace,
//		}
//
//		By("Deploying a pod with proper annotations successfully")
//		daemonSetResource = &appsv1.DaemonSet{
//			TypeMeta: metav1.TypeMeta{
//				Kind:       daemonSetKind,
//				APIVersion: daemonSetAPIVersion,
//			},
//			ObjectMeta: metav1.ObjectMeta{
//				Name:      daemonSetKey.Name,
//				Namespace: daemonSetKey.Namespace,
//				Annotations: map[string]string{
//					op.ItemPathAnnotation: item1.Path,
//					op.NameAnnotation:     item1.Name,
//				},
//			},
//			Spec: appsv1.DaemonSetSpec{
//				Template: v1.PodTemplateSpec{
//					ObjectMeta: metav1.ObjectMeta{
//						Labels: map[string]string{"app": daemonSetName},
//					},
//					Spec: v1.PodSpec{
//						Containers: []v1.Container{
//							{
//								Name:            daemonSetName,
//								Image:           "eu.gcr.io/kyma-project/example/http-db-service:0.0.6",
//								ImagePullPolicy: "IfNotPresent",
//							},
//						},
//					},
//				},
//				Selector: &metav1.LabelSelector{
//					MatchLabels: map[string]string{"app": daemonSetName},
//				},
//			},
//		}
//		Expect(k8sClient.Create(ctx, daemonSetResource)).Should(Succeed())
//
//		By("Creating the K8s secret successfully")
//		time.Sleep(time.Millisecond * 100)
//		Eventually(func() bool {
//			err := k8sClient.Get(ctx, secretKey, createdSecret)
//			if err != nil {
//				return false
//			}
//			return true
//		}, timeout, interval).Should(BeTrue())
//		Expect(createdSecret.Data).Should(Equal(item1.SecretData))
//	}
//
//	cleanK8sResources := func() {
//		// failed test runs that don't clean up leave resources behind.
//		err := k8sClient.DeleteAllOf(context.Background(), &onepasswordv1.OnePasswordItem{}, client.InNamespace(namespace))
//		Expect(err).ToNot(HaveOccurred())
//
//		err = k8sClient.DeleteAllOf(context.Background(), &v1.Secret{}, client.InNamespace(namespace))
//		Expect(err).ToNot(HaveOccurred())
//
//		err = k8sClient.DeleteAllOf(context.Background(), &appsv1.DaemonSet{}, client.InNamespace(namespace))
//		Expect(err).ToNot(HaveOccurred())
//	}
//
//	mockGetItemFunc := func() {
//		mocks.DoGetItemFunc = func(uuid string, vaultUUID string) (*onepassword.Item, error) {
//			item := onepassword.Item{}
//			item.Fields = []*onepassword.ItemField{}
//			for k, v := range item1.Data {
//				item.Fields = append(item.Fields, &onepassword.ItemField{Label: k, Value: v})
//			}
//			item.Version = item1.Version
//			item.Vault.ID = vaultUUID
//			item.ID = uuid
//			return &item, nil
//		}
//	}
//
//	BeforeEach(func() {
//		cleanK8sResources()
//		mockGetItemFunc()
//		time.Sleep(time.Second) // TODO: can we achieve that with ginkgo?
//		makeDaemonSet()
//	})
//
//	Context("DaemonSet with secrets from 1Password", func() {
//		It("Should delete secret if daemonSet is deleted", func() {
//			By("Deleting the pod")
//			Eventually(func() error {
//				f := &appsv1.DaemonSet{}
//				err := k8sClient.Get(ctx, daemonSetKey, f)
//				if err != nil {
//					return err
//				}
//				return k8sClient.Delete(ctx, f)
//			}, timeout, interval).Should(Succeed())
//
//			Eventually(func() error {
//				f := &appsv1.DaemonSet{}
//				return k8sClient.Get(ctx, daemonSetKey, f)
//			}, timeout, interval).ShouldNot(Succeed())
//
//			Eventually(func() error {
//				f := &v1.Secret{}
//				return k8sClient.Get(ctx, secretKey, f)
//			}, timeout, interval).ShouldNot(Succeed())
//		})
//
//		It("Should update existing K8s Secret using daemonSet", func() {
//			By("Updating secret")
//			mocks.DoGetItemFunc = func(uuid string, vaultUUID string) (*onepassword.Item, error) {
//				item := onepassword.Item{}
//				item.Fields = []*onepassword.ItemField{}
//				for k, v := range item2.Data {
//					item.Fields = append(item.Fields, &onepassword.ItemField{Label: k, Value: v})
//				}
//				item.Version = item2.Version
//				item.Vault.ID = vaultUUID
//				item.ID = uuid
//				return &item, nil
//			}
//			Eventually(func() error {
//				updatedDaemonSet := &appsv1.DaemonSet{
//					TypeMeta: metav1.TypeMeta{
//						Kind:       daemonSetKind,
//						APIVersion: daemonSetAPIVersion,
//					},
//					ObjectMeta: metav1.ObjectMeta{
//						Name:      daemonSetKey.Name,
//						Namespace: daemonSetKey.Namespace,
//						Annotations: map[string]string{
//							op.ItemPathAnnotation: item2.Path,
//							op.NameAnnotation:     item1.Name,
//						},
//					},
//					Spec: appsv1.DaemonSetSpec{
//						Template: v1.PodTemplateSpec{
//							ObjectMeta: metav1.ObjectMeta{
//								Labels: map[string]string{"app": daemonSetName},
//							},
//							Spec: v1.PodSpec{
//								Containers: []v1.Container{
//									{
//										Name:            daemonSetName,
//										Image:           "eu.gcr.io/kyma-project/example/http-db-service:0.0.6",
//										ImagePullPolicy: "IfNotPresent",
//									},
//								},
//							},
//						},
//						Selector: &metav1.LabelSelector{
//							MatchLabels: map[string]string{"app": daemonSetName},
//						},
//					},
//				}
//				err := k8sClient.Update(ctx, updatedDaemonSet)
//				if err != nil {
//					return err
//				}
//				return nil
//			}, timeout, interval).Should(Succeed())
//
//			// TODO: can we achieve the same without sleep?
//			time.Sleep(time.Millisecond * 10)
//			By("Reading updated K8s secret")
//			updatedSecret := &v1.Secret{}
//			Eventually(func() bool {
//				err := k8sClient.Get(ctx, secretKey, updatedSecret)
//				if err != nil {
//					return false
//				}
//				return true
//			}, timeout, interval).Should(BeTrue())
//			Expect(updatedSecret.Data).Should(Equal(item2.SecretData))
//		})
//
//		It("Should not update secret if Annotations have not changed", func() {
//			By("Updating secret without changing annotations")
//			Eventually(func() error {
//				updatedDaemonSet := &appsv1.DaemonSet{
//					TypeMeta: metav1.TypeMeta{
//						Kind:       daemonSetKind,
//						APIVersion: daemonSetAPIVersion,
//					},
//					ObjectMeta: metav1.ObjectMeta{
//						Name:      daemonSetKey.Name,
//						Namespace: daemonSetKey.Namespace,
//						Annotations: map[string]string{
//							op.ItemPathAnnotation: item1.Path,
//							op.NameAnnotation:     item1.Name,
//						},
//					},
//					Spec: appsv1.DaemonSetSpec{
//						Template: v1.PodTemplateSpec{
//							ObjectMeta: metav1.ObjectMeta{
//								Labels: map[string]string{"app": daemonSetName},
//							},
//							Spec: v1.PodSpec{
//								Containers: []v1.Container{
//									{
//										Name:            daemonSetName,
//										Image:           "eu.gcr.io/kyma-project/example/http-db-service:0.0.6",
//										ImagePullPolicy: "IfNotPresent",
//									},
//								},
//							},
//						},
//						Selector: &metav1.LabelSelector{
//							MatchLabels: map[string]string{"app": daemonSetName},
//						},
//					},
//				}
//				err := k8sClient.Update(ctx, updatedDaemonSet)
//				if err != nil {
//					return err
//				}
//				return nil
//			}, timeout, interval).Should(Succeed())
//
//			// TODO: can we achieve the same without sleep?
//			time.Sleep(time.Millisecond * 10)
//			By("Reading updated K8s secret")
//			updatedSecret := &v1.Secret{}
//			Eventually(func() bool {
//				err := k8sClient.Get(ctx, secretKey, updatedSecret)
//				if err != nil {
//					return false
//				}
//				return true
//			}, timeout, interval).Should(BeTrue())
//			Expect(updatedSecret.Data).Should(Equal(item1.SecretData))
//		})
//
//		It("Should not delete secret created via daemonSet if it's used in another container", func() {
//			By("Creating another POD with created secret")
//			anotherDaemonSetKey := types.NamespacedName{
//				Name:      "other-daemonSet",
//				Namespace: namespace,
//			}
//			Eventually(func() error {
//				anotherDaemonSet := &appsv1.DaemonSet{
//					TypeMeta: metav1.TypeMeta{
//						Kind:       daemonSetKind,
//						APIVersion: daemonSetAPIVersion,
//					},
//					ObjectMeta: metav1.ObjectMeta{
//						Name:      anotherDaemonSetKey.Name,
//						Namespace: anotherDaemonSetKey.Namespace,
//					},
//					Spec: appsv1.DaemonSetSpec{
//						Template: v1.PodTemplateSpec{
//							ObjectMeta: metav1.ObjectMeta{
//								Labels: map[string]string{"app": anotherDaemonSetKey.Name},
//							},
//							Spec: v1.PodSpec{
//								Containers: []v1.Container{
//									{
//										Name:            anotherDaemonSetKey.Name,
//										Image:           "eu.gcr.io/kyma-project/example/http-db-service:0.0.6",
//										ImagePullPolicy: "IfNotPresent",
//										Env: []v1.EnvVar{
//											{
//												Name: anotherDaemonSetKey.Name,
//												ValueFrom: &v1.EnvVarSource{
//													SecretKeyRef: &v1.SecretKeySelector{
//														LocalObjectReference: v1.LocalObjectReference{
//															Name: secretKey.Name,
//														},
//														Key: "password",
//													},
//												},
//											},
//										},
//									},
//								},
//							},
//						},
//						Selector: &metav1.LabelSelector{
//							MatchLabels: map[string]string{"app": anotherDaemonSetKey.Name},
//						},
//					},
//				}
//				err := k8sClient.Create(ctx, anotherDaemonSet)
//				if err != nil {
//					return err
//				}
//				return nil
//			}, timeout, interval).Should(Succeed())
//
//			By("Deleting the pod")
//			Eventually(func() error {
//				f := &appsv1.DaemonSet{}
//				err := k8sClient.Get(ctx, daemonSetKey, f)
//				if err != nil {
//					return err
//				}
//				return k8sClient.Delete(ctx, f)
//			}, timeout, interval).Should(Succeed())
//
//			Eventually(func() error {
//				f := &v1.Secret{}
//				return k8sClient.Get(ctx, secretKey, f)
//			}, timeout, interval).Should(Succeed())
//		})
//
//		It("Should not delete secret created via daemonSet if it's used in another volume", func() {
//			By("Creating another POD with created secret")
//			anotherDaemonSetKey := types.NamespacedName{
//				Name:      "other-daemonSet",
//				Namespace: namespace,
//			}
//			Eventually(func() error {
//				anotherDaemonSet := &appsv1.DaemonSet{
//					TypeMeta: metav1.TypeMeta{
//						Kind:       daemonSetKind,
//						APIVersion: daemonSetAPIVersion,
//					},
//					ObjectMeta: metav1.ObjectMeta{
//						Name:      anotherDaemonSetKey.Name,
//						Namespace: anotherDaemonSetKey.Namespace,
//					},
//					Spec: appsv1.DaemonSetSpec{
//						Template: v1.PodTemplateSpec{
//							ObjectMeta: metav1.ObjectMeta{
//								Labels: map[string]string{"app": anotherDaemonSetKey.Name},
//							},
//							Spec: v1.PodSpec{
//								Volumes: []v1.Volume{
//									{
//										Name: anotherDaemonSetKey.Name,
//										VolumeSource: v1.VolumeSource{
//											Secret: &v1.SecretVolumeSource{
//												SecretName: secretKey.Name,
//											},
//										},
//									},
//								},
//								Containers: []v1.Container{
//									{
//										Name:            anotherDaemonSetKey.Name,
//										Image:           "eu.gcr.io/kyma-project/example/http-db-service:0.0.6",
//										ImagePullPolicy: "IfNotPresent",
//									},
//								},
//							},
//						},
//						Selector: &metav1.LabelSelector{
//							MatchLabels: map[string]string{"app": anotherDaemonSetKey.Name},
//						},
//					},
//				}
//				err := k8sClient.Create(ctx, anotherDaemonSet)
//				if err != nil {
//					return err
//				}
//				return nil
//			}, timeout, interval).Should(Succeed())
//
//			By("Deleting the pod")
//			Eventually(func() error {
//				f := &appsv1.DaemonSet{}
//				err := k8sClient.Get(ctx, daemonSetKey, f)
//				if err != nil {
//					return err
//				}
//				return k8sClient.Delete(ctx, f)
//			}, timeout, interval).Should(Succeed())
//
//			Eventually(func() error {
//				f := &v1.Secret{}
//				return k8sClient.Get(ctx, secretKey, f)
//			}, timeout, interval).Should(Succeed())
//		})
//	})
//})
